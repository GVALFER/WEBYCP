package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GVALFER/WEBYCP/internal/httpapi/spec"
)

func checkTaskAPI(t *testing.T, api http.Handler, cookie *http.Cookie, csrf, accountID string) {
	t.Helper()
	request := func(method, path, body string, token bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.AddCookie(cookie)
		if token {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		response := httptest.NewRecorder()
		api.ServeHTTP(response, r)
		return response
	}
	body := `{"accountId":"` + accountID + `","name":"Hourly task","schedule":"0 * * * *","command":"echo private-command-content","schedulerDriver":"crontab","kind":"command","enabled":true}`
	if r := request(http.MethodPost, "/api/v1/scheduled-tasks", body, false); r.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF status = %d", r.Code)
	}
	if r := request(http.MethodPost, "/api/v1/scheduled-tasks", strings.Replace(body, `"kind":"command"`, `"kind":"http"`, 1), true); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unsupported kind status = %d: %s", r.Code, r.Body.String())
	}
	if r := request(http.MethodPost, "/api/v1/scheduled-tasks", strings.TrimSuffix(body, "}")+`,"runAs":"root"}`, true); r.Code != http.StatusBadRequest {
		t.Fatalf("runAs override status = %d", r.Code)
	}
	created := request(http.MethodPost, "/api/v1/scheduled-tasks", body, true)
	if created.Code != http.StatusAccepted {
		t.Fatalf("create = %d: %s", created.Code, created.Body.String())
	}
	var value publicapi.ScheduledTaskResponse
	if err := json.Unmarshal(created.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	if value.ScheduledTask.Kind != publicapi.TaskKind("command") || value.Job.Kind != "task.sync" {
		t.Fatalf("response = %+v", value)
	}
	path := "/api/v1/scheduled-tasks/" + value.ScheduledTask.Id
	otherAccount := strings.Replace(body, accountID, "0123456789abcdef0123456789abcdef", 1)
	if r := request(http.MethodPatch, path, otherAccount, true); r.Code != http.StatusForbidden {
		t.Fatalf("account reassignment = %d", r.Code)
	}
	updated := request(http.MethodPatch, path, strings.Replace(body, `"enabled":true`, `"enabled":false`, 1), true)
	if updated.Code != http.StatusAccepted || !strings.Contains(updated.Body.String(), `"enabled":false`) {
		t.Fatalf("update = %d: %s", updated.Code, updated.Body.String())
	}
	listed := request(http.MethodGet, "/api/v1/audit-events?jobId="+value.Job.Id, "", false)
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), `"action":"task.create"`) || !strings.Contains(listed.Body.String(), `"jobId":"`+value.Job.Id+`"`) || strings.Contains(listed.Body.String(), "private-command-content") {
		t.Fatalf("audit = %d: %s", listed.Code, listed.Body.String())
	}
	if r := request(http.MethodGet, "/api/v1/audit-events?jobId=invalid", "", false); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid job filter = %d", r.Code)
	}
	if r := request(http.MethodDelete, path, "", true); r.Code != http.StatusAccepted {
		t.Fatalf("delete = %d: %s", r.Code, r.Body.String())
	}
	if r := request(http.MethodGet, "/api/v1/cron-jobs", "", false); r.Code != http.StatusNotFound {
		t.Fatalf("obsolete route = %d", r.Code)
	}
}
