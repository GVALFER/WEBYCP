package httpapi

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/certificates"
	publicapi "github.com/GVALFER/WEBYCP/internal/httpapi/spec"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/websites"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func (h *handler) listCertificates(w http.ResponseWriter, r *http.Request, session auth.Session) {
	query, ok := requestPage(w, r)
	if !ok {
		return
	}
	page, err := h.options.Certificates.CertificatePage(
		r.Context(), session.User.ID, session.User.Role == "admin", r.URL.Query().Get("kind"), query,
	)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response := publicapi.CertificateListResponse{
		Items:      make([]publicapi.Certificate, 0, len(page.Items)),
		Pagination: paginationResponse(page.Query, page.Total),
	}
	for _, item := range page.Items {
		response.Items = append(response.Items, certificateResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) issueWebsiteCertificate(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.IssueCertificateRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	value, job, err := h.options.Certificates.IssueWebsite(r.Context(), r.PathValue("websiteId"), string(request.Email), session.User.ID, session.User.Role == "admin")
	h.recordJobMutation(r, session.User.ID, "certificate.issue", "certificate", value.ID, job.ID, err)
	if err != nil {
		h.writeCertificateError(w, r, err)
		return
	}
	writeCertificateJob(w, value, job)
}

func (h *handler) issuePanelCertificate(w http.ResponseWriter, r *http.Request, session auth.Session) {
	if session.User.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "Administrator access is required")
		return
	}
	var request publicapi.IssuePanelCertificateRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	value, job, err := h.options.Certificates.IssuePanel(r.Context(), request.Hostname, string(request.Email), session.User.ID)
	h.recordJobMutation(r, session.User.ID, "certificate.panel_issue", "certificate", value.ID, job.ID, err)
	if err != nil {
		h.writeCertificateError(w, r, err)
		return
	}
	writeCertificateJob(w, value, job)
}

func (h *handler) renewCertificate(w http.ResponseWriter, r *http.Request, session auth.Session) {
	value, job, err := h.options.Certificates.Renew(r.Context(), r.PathValue("certificateId"), session.User.ID, session.User.Role == "admin")
	h.recordJobMutation(r, session.User.ID, "certificate.renew", "certificate", r.PathValue("certificateId"), job.ID, err)
	if err != nil {
		h.writeCertificateError(w, r, err)
		return
	}
	writeCertificateJob(w, value, job)
}

func (h *handler) setCertificate(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.UpdateCertificateRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	value, job, err := h.options.Certificates.SetRedirect(r.Context(), r.PathValue("certificateId"), session.User.ID, session.User.Role == "admin", request.RedirectHttps)
	h.recordJobMutation(r, session.User.ID, "certificate.redirect", "certificate", r.PathValue("certificateId"), job.ID, err)
	if err != nil {
		h.writeCertificateError(w, r, err)
		return
	}
	writeCertificateJob(w, value, job)
}

func (h *handler) writeCertificateError(w http.ResponseWriter, r *http.Request, err error) {
	if writeValidationError(w, err) {
		return
	}
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, "not_found", "Certificate or website not found")
	case errors.Is(err, accounts.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "Account access is required")
	case errors.Is(err, websites.ErrWebsiteInactive):
		writeError(w, http.StatusConflict, "website_inactive", "The website must be active")
	case errors.Is(err, certificates.ErrBusy):
		writeError(w, http.StatusConflict, "resource_busy", "A certificate operation is already pending")
	default:
		h.internalError(w, r, err)
	}
}

func certificateResponse(value certificates.Certificate) publicapi.Certificate {
	response := publicapi.Certificate{
		Id: value.ID, NodeId: value.NodeID, Kind: publicapi.CertificateKind(value.Kind),
		Name: value.Name, Names: value.Names, Email: openapi_types.Email(value.Email),
		Status: publicapi.CertificateStatus(value.Status), RedirectHttps: value.RedirectHTTPS,
		ExpiresAt: value.ExpiresAt, Error: value.Error, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
	if value.WebsiteID != "" {
		response.WebsiteId = &value.WebsiteID
	}
	return response
}

func writeCertificateJob(w http.ResponseWriter, value certificates.Certificate, job jobs.Job) {
	httpx.WriteJSON(w, http.StatusAccepted, publicapi.CertificateJobResponse{Certificate: certificateResponse(value), Job: jobResponse(job)})
}
