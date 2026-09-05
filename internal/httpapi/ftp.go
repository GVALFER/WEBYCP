package httpapi

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/ftp"
	publicapi "github.com/GVALFER/WEBYCP/internal/httpapi/spec"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/jobs"
)

func (h *handler) listFTPAccounts(w http.ResponseWriter, r *http.Request, session auth.Session) {
	query, ok := requestPage(w, r)
	if !ok {
		return
	}
	page, err := h.options.FTP.Page(r.Context(), session.User.ID, session.User.Role == "admin", query)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response := publicapi.FTPAccountListResponse{
		Items:      make([]publicapi.FTPAccount, 0, len(page.Items)),
		Pagination: paginationResponse(page.Query, page.Total),
	}
	for _, item := range page.Items {
		response.Items = append(response.Items, ftpResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) createFTPAccount(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.CreateFTPAccountRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil || request.Password == nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Provide a valid FTP account and password")
		return
	}
	value, job, err := h.options.FTP.Create(r.Context(), request.AccountId, request.Username,
		*request.Password, session.User.ID, session.User.Role == "admin", request.Enabled)
	h.recordJobMutation(r, session.User.ID, "ftp.create", "ftp_account", value.ID, job.ID, err)
	if err != nil {
		h.writeFTPError(w, r, err)
		return
	}
	writeFTP(w, value, job)
}

func (h *handler) updateFTPAccount(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.UpdateFTPAccountRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	id := r.PathValue("ftpAccountId")
	value, job, err := h.options.FTP.Update(r.Context(), id, session.User.ID, session.User.Role == "admin",
		request.Username, request.Password, request.Enabled)
	h.recordJobMutation(r, session.User.ID, "ftp.update", "ftp_account", id, job.ID, err)
	if err != nil {
		h.writeFTPError(w, r, err)
		return
	}
	writeFTP(w, value, job)
}

func (h *handler) deleteFTPAccount(w http.ResponseWriter, r *http.Request, session auth.Session) {
	id := r.PathValue("ftpAccountId")
	job, err := h.options.FTP.Delete(r.Context(), id, session.User.ID, session.User.Role == "admin")
	h.recordJobMutation(r, session.User.ID, "ftp.delete", "ftp_account", id, job.ID, err)
	if err != nil {
		h.writeFTPError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, jobResponse(job))
}

func (h *handler) writeFTPError(w http.ResponseWriter, r *http.Request, err error) {
	if writeValidationError(w, err) || writeLimitError(w, err) {
		return
	}
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, "not_found", "FTP account or hosting account not found")
	case errors.Is(err, accounts.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "Account access is required")
	case errors.Is(err, accounts.ErrBusy):
		writeError(w, http.StatusConflict, "account_inactive", "The hosting account is not ready for this change")
	case errors.Is(err, ftp.ErrBusy):
		writeError(w, http.StatusConflict, "ftp_busy", "Wait for the pending FTP job for this hosting account")
	case errors.Is(err, ftp.ErrNameExists):
		writeError(w, http.StatusConflict, "ftp_username_exists", "This FTP username is already in use on the server")
	case errors.Is(err, ftp.ErrDeleting):
		writeError(w, http.StatusConflict, "ftp_deleting", "Retry deletion to finish revoking this FTP account")
	default:
		h.internalError(w, r, err)
	}
}

func ftpResponse(value ftp.Account) publicapi.FTPAccount {
	return publicapi.FTPAccount{
		Id: value.ID, AccountId: value.AccountID, NodeId: value.NodeID, Username: value.Username,
		AccountName: value.AccountName, AccountStatus: value.AccountStatus, Home: "/home/" + value.SystemUser,
		Enabled: value.Enabled, Deleting: value.Deleting, Status: publicapi.FTPAccountStatus(value.Status),
		CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func writeFTP(w http.ResponseWriter, value ftp.Account, job jobs.Job) {
	httpx.WriteJSON(w, http.StatusAccepted, publicapi.FTPAccountResponse{FtpAccount: ftpResponse(value), Job: jobResponse(job)})
}
