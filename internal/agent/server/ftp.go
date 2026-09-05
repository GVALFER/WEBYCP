package server

import (
	"errors"
	"net/http"

	"github.com/GVALFER/WEBYCP/internal/agent/ftp"
	"github.com/GVALFER/WEBYCP/internal/agent/hostuser"
	agentapi "github.com/GVALFER/WEBYCP/internal/agent/protocol"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

func ftpHandler(driver ftp.Driver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request agentapi.SyncFTPRequest
		if err := httpx.DecodeJSON(w, r, &request); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, agentapi.ErrorResponse{Code: "invalid_json", Message: "The request body is invalid"})
			return
		}
		if hostuser.ValidateNames(request.AccountId, request.SystemUser) != nil || request.Entries == nil || len(request.Entries) > 100 {
			httpx.WriteJSON(w, http.StatusUnprocessableEntity, agentapi.ErrorResponse{Code: "validation_error", Message: "The FTP account request is invalid"})
			return
		}
		if driver == nil {
			httpx.WriteJSON(w, http.StatusServiceUnavailable, agentapi.ErrorResponse{Code: "ftp_unavailable", Message: "The FTP driver is not configured"})
			return
		}
		entries := make([]ftp.Entry, 0, len(request.Entries))
		for _, entry := range request.Entries {
			entries = append(entries, ftp.Entry{ID: entry.Id, Username: entry.Username, PasswordHash: entry.PasswordHash, Enabled: entry.Enabled})
		}
		if err := driver.Sync(r.Context(), request.AccountId, request.SystemUser, entries); err != nil {
			var invalid *validate.Error
			if errors.As(err, &invalid) {
				httpx.WriteJSON(w, http.StatusUnprocessableEntity, agentapi.ErrorResponse{Code: "validation_error", Message: invalid.Message})
			} else {
				// Authentication state and subprocess output must not enter logs.
				httpx.WriteJSON(w, http.StatusInternalServerError, agentapi.ErrorResponse{Code: "ftp_failed", Message: "FTP access could not be synchronized"})
			}
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
