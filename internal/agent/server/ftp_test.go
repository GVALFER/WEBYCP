package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GVALFER/WEBYCP/internal/agent/ftp"
	agentapi "github.com/GVALFER/WEBYCP/internal/agent/protocol"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

type ftpDriver struct {
	account, user string
	entries       []ftp.Entry
	err           error
}

func (d *ftpDriver) Sync(_ context.Context, account, user string, entries []ftp.Entry) error {
	d.account, d.user, d.entries = account, user, entries
	return d.err
}

func TestFTPProtocol(t *testing.T) {
	request := agentapi.SyncFTPRequest{
		AccountId: "0123456789abcdef0123456789abcdef", SystemUser: "wcp_0123456789ab",
		Entries: []agentapi.FTPEntry{{Id: "abcdef0123456789abcdef0123456789", Username: "customer", PasswordHash: "private-hash", Enabled: true}},
	}
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	for _, test := range []struct {
		name string
		body string
		err  error
		want int
	}{
		{"sync", body, nil, http.StatusNoContent},
		{"root", strings.Replace(body, "wcp_0123456789ab", "root", 1), nil, http.StatusUnprocessableEntity},
		{"arbitrary jail", strings.Replace(body, `"entries":`, `"home":"/root","entries":`, 1), nil, http.StatusBadRequest},
		{"plaintext field", strings.Replace(body, `"passwordHash":`, `"password":"plaintext","passwordHash":`, 1), nil, http.StatusBadRequest},
		{"missing entries", `{"accountId":"0123456789abcdef0123456789abcdef","systemUser":"wcp_0123456789ab"}`, nil, http.StatusUnprocessableEntity},
		{"remove all", `{"accountId":"0123456789abcdef0123456789abcdef","systemUser":"wcp_0123456789ab","entries":[]}`, nil, http.StatusNoContent},
		{"validation", body, &validate.Error{Field: "username", Message: "Invalid FTP username"}, http.StatusUnprocessableEntity},
		{"failure", body, errors.New("private-hash plaintext private-file"), http.StatusInternalServerError},
	} {
		t.Run(test.name, func(t *testing.T) {
			driver := &ftpDriver{err: test.err}
			var logs bytes.Buffer
			response := httptest.NewRecorder()
			New(Options{FTP: driver, Logger: slog.New(slog.NewTextHandler(&logs, nil))}).ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/agent/v1/ftp-accounts", strings.NewReader(test.body)))
			if response.Code != test.want {
				t.Fatalf("status = %d, want %d", response.Code, test.want)
			}
			for _, secret := range []string{"private-hash", "plaintext", "private-file"} {
				if strings.Contains(logs.String()+response.Body.String(), secret) {
					t.Fatal("FTP request secrets entered logs or response")
				}
			}
			if test.want < 300 && (driver.account != request.AccountId || driver.user != request.SystemUser) {
				t.Fatal("incorrect account mapping")
			}
			if test.name == "sync" && (len(driver.entries) != 1 || driver.entries[0].PasswordHash != "private-hash" || !driver.entries[0].Enabled) {
				t.Fatal("FTP entry changed across the protocol")
			}
			if (test.want == 400 || test.name == "root") && driver.account != "" {
				t.Fatal("invalid request reached the driver")
			}
		})
	}
	response := httptest.NewRecorder()
	New(Options{}).ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/agent/v1/ftp-accounts", strings.NewReader(body)))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatal("missing driver was not reported")
	}
}
