package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GVALFER/WEBYCP/internal/agent/scheduler"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

type taskDriver struct {
	account, user string
	entries       []scheduler.Entry
	err           error
}

func (d *taskDriver) Sync(_ context.Context, account, user string, entries []scheduler.Entry) error {
	d.account, d.user, d.entries = account, user, entries
	return d.err
}

func TestScheduledTaskProtocol(t *testing.T) {
	const body = `{"accountId":"0123456789abcdef0123456789abcdef","systemUser":"wcp_0123456789ab","entries":[{"id":"abcdef0123456789abcdef0123456789","kind":"command","schedule":"0 * * * *","command":"/usr/bin/true"}]}`
	for _, test := range []struct {
		name string
		body string
		err  error
		want int
	}{
		{name: "sync", body: body, want: http.StatusNoContent},
		{name: "remove all", body: `{"accountId":"0123456789abcdef0123456789abcdef","systemUser":"wcp_0123456789ab","entries":[]}`, want: http.StatusNoContent},
		{name: "root override", body: strings.Replace(body, "wcp_0123456789ab", "root", 1), want: http.StatusUnprocessableEntity},
		{name: "unknown field", body: strings.Replace(body, `"entries":`, `"runAs":"root","entries":`, 1), want: http.StatusBadRequest},
		{name: "invalid task", body: body, err: &validate.Error{Field: "kind", Message: "Unsupported task kind"}, want: http.StatusUnprocessableEntity},
		{name: "driver failure", body: body, err: errors.New("private driver detail"), want: http.StatusInternalServerError},
	} {
		t.Run(test.name, func(t *testing.T) {
			driver := &taskDriver{err: test.err}
			response := httptest.NewRecorder()
			New(Options{Tasks: driver}).ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/agent/v1/scheduled-tasks", strings.NewReader(test.body)))
			if response.Code != test.want || strings.Contains(response.Body.String(), "private driver detail") {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			if test.name == "sync" && (driver.account != "0123456789abcdef0123456789abcdef" || driver.user != "wcp_0123456789ab" || len(driver.entries) != 1 || driver.entries[0].Kind != "command") {
				t.Fatalf("driver = %+v", driver)
			}
			if test.name == "root override" && driver.account != "" {
				t.Fatal("invalid identity reached the driver")
			}
		})
	}
	response := httptest.NewRecorder()
	New(Options{Tasks: &taskDriver{}}).ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/agent/v1/cron", strings.NewReader(body)))
	if response.Code != http.StatusNotFound {
		t.Fatalf("obsolete route status = %d", response.Code)
	}
}
