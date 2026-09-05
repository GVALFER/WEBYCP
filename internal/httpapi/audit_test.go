package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GVALFER/WEBYCP/internal/auth"
)

func TestAuditRequiresAdministrator(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/audit-events", nil)
	response := httptest.NewRecorder()
	New(Options{}).ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous status = %d", response.Code)
	}
	response = httptest.NewRecorder()
	h := &handler{}
	h.listAuditEvents(response, request, auth.Session{User: auth.User{Role: "user"}})
	if response.Code != http.StatusForbidden {
		t.Fatalf("user status = %d", response.Code)
	}
	for _, route := range []authedHandler{h.listJobs, h.getJob} {
		response := httptest.NewRecorder()
		route(response, request, auth.Session{User: auth.User{Role: "user"}})
		if response.Code != http.StatusForbidden {
			t.Fatalf("job user status = %d", response.Code)
		}
	}
}
