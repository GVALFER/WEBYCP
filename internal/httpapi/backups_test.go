package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GVALFER/WEBYCP/internal/backupfmt"
	publicapi "github.com/GVALFER/WEBYCP/internal/httpapi/spec"
)

func TestBackupArchiveErrors(t *testing.T) {
	for _, test := range []struct {
		err  error
		code string
	}{
		{backupfmt.ErrVersion, "backup_version"},
		{backupfmt.ErrInvalid, "backup_invalid"},
	} {
		t.Run(test.code, func(t *testing.T) {
			response := httptest.NewRecorder()
			h := &handler{}
			h.writeBackupError(response, httptest.NewRequest(http.MethodGet, "/", nil), fmt.Errorf("private path: %w", test.err))
			var body publicapi.ErrorResponse
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusUnprocessableEntity || body.Code != test.code || body.Message != test.err.Error() || strings.Contains(response.Body.String(), "private") {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
}
