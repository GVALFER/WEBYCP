package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	publicapi "github.com/GVALFER/WEBYCP/internal/httpapi/spec"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite"
)

func checkFTPAPI(t *testing.T, api http.Handler, store *sqlite.Store, cookie *http.Cookie, csrf, accountID string) {
	t.Helper()
	request := func(method, path, body string, authenticated, token bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if authenticated {
			r.AddCookie(cookie)
		}
		if token {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		response := httptest.NewRecorder()
		api.ServeHTTP(response, r)
		return response
	}
	body := `{"accountId":"` + accountID + `","username":"ftp.owner","password":"private FTP test password","enabled":true}`
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete} {
		path := "/api/v1/ftp-accounts"
		if method == http.MethodPatch || method == http.MethodDelete {
			path += "/0123456789abcdef0123456789abcdef"
		}
		if r := request(method, path, body, false, false); r.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated %s = %d", method, r.Code)
		}
		if method != http.MethodGet {
			if r := request(method, path, body, true, false); r.Code != http.StatusForbidden {
				t.Fatalf("missing CSRF %s = %d", method, r.Code)
			}
		}
	}
	for _, field := range []string{`"home":"/root"`, `"passwordHash":"private"`, `"systemUser":"root"`} {
		r := request(http.MethodPost, "/api/v1/ftp-accounts", strings.TrimSuffix(body, "}")+","+field+"}", true, true)
		if r.Code != http.StatusBadRequest {
			t.Fatalf("privileged override accepted: %d", r.Code)
		}
	}
	if r := request(http.MethodPost, "/api/v1/ftp-accounts", strings.Replace(body, "private FTP test password", "short", 1), true, true); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("weak password = %d", r.Code)
	}
	created := request(http.MethodPost, "/api/v1/ftp-accounts", body, true, true)
	if created.Code != http.StatusAccepted {
		t.Fatalf("FTP create = %d: %s", created.Code, created.Body.String())
	}
	var value publicapi.FTPAccountResponse
	if err := json.Unmarshal(created.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	if value.FtpAccount.Home == "" || value.FtpAccount.Username != "ftp.owner" || value.Job.Kind != "ftp.sync" {
		t.Fatal("invalid public FTP response")
	}
	path := "/api/v1/ftp-accounts/" + value.FtpAccount.Id
	if r := request(http.MethodPatch, path, `{"enabled":false}`, true, true); r.Code != http.StatusConflict {
		t.Fatalf("pending change = %d", r.Code)
	}
	if r := request(http.MethodPatch, path, `{"accountId":"`+accountID+`"}`, true, true); r.Code != http.StatusBadRequest {
		t.Fatalf("account reassignment = %d", r.Code)
	}
	if r := request(http.MethodPatch, path, `{}`, true, true); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("empty patch = %d", r.Code)
	}
	stored, err := store.FTP(context.Background(), value.FtpAccount.Id)
	if err != nil {
		t.Fatal(err)
	}
	for _, response := range []*httptest.ResponseRecorder{
		created,
		request(http.MethodGet, "/api/v1/ftp-accounts?page=1&size=10", "", true, false),
		request(http.MethodGet, "/api/v1/jobs/"+value.Job.Id, "", true, false),
		request(http.MethodGet, "/api/v1/audit-events?jobId="+value.Job.Id, "", true, false),
	} {
		for _, secret := range []string{"private FTP test password", stored.PasswordHash, "passwordHash", "password_hash"} {
			if strings.Contains(response.Body.String(), secret) {
				t.Fatal("public response leaked FTP credential")
			}
		}
	}
	audit := request(http.MethodGet, "/api/v1/audit-events?jobId="+value.Job.Id, "", true, false)
	if !strings.Contains(audit.Body.String(), `"action":"ftp.create"`) {
		t.Fatal("missing FTP audit correlation")
	}
	if r := request(http.MethodGet, "/api/v1/ftp-accounts?page=bad", "", true, false); r.Code != http.StatusBadRequest {
		t.Fatalf("invalid pagination = %d", r.Code)
	}
	// The Agent's successful/failed execution is covered across the Unix socket
	// in ftp/service_test.go. Release this test's queued operation before PATCH.
	for {
		job, err := store.ClaimJob(context.Background(), time.Now().UTC())
		if err != nil {
			t.Fatal(err)
		}
		if job.ID == value.Job.Id {
			break
		}
		if err := store.CompleteJob(context.Background(), job.ID, time.Now().UTC()); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.FailJob(context.Background(), value.Job.Id, "test synchronization unavailable", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := store.FinishFTP(context.Background(), accountID, true); err != nil {
		t.Fatal(err)
	}
	updated := request(http.MethodPatch, path, `{"username":"ftp.renamed","enabled":false}`, true, true)
	if updated.Code != http.StatusAccepted || !strings.Contains(updated.Body.String(), `"enabled":false`) {
		t.Fatalf("FTP patch = %d: %s", updated.Code, updated.Body.String())
	}
}
