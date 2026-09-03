package httpapi

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/certificates"
	"github.com/GVALFER/WEBYCP/internal/domains"
	"github.com/GVALFER/WEBYCP/internal/httpapi/spec"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func (h *handler) listCertificates(w http.ResponseWriter, r *http.Request, session auth.Session) {
	items, err := h.options.Certificates.Certificates(r.Context(), session.User.ID, session.User.Role == "admin")
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response := publicapi.CertificateListResponse{Items: make([]publicapi.Certificate, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, certificateResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) issueDomainCertificate(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.IssueCertificateRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	value, job, err := h.options.Certificates.IssueDomain(r.Context(), r.PathValue("domainId"), string(request.Email), session.User.ID, session.User.Role == "admin")
	h.recordMutation(r, session.User.ID, "certificate.issue", "certificate", value.ID, err)
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
	h.recordMutation(r, session.User.ID, "certificate.panel_issue", "certificate", value.ID, err)
	if err != nil {
		h.writeCertificateError(w, r, err)
		return
	}
	writeCertificateJob(w, value, job)
}

func (h *handler) renewCertificate(w http.ResponseWriter, r *http.Request, session auth.Session) {
	value, job, err := h.options.Certificates.Renew(r.Context(), r.PathValue("certificateId"), session.User.ID, session.User.Role == "admin")
	h.recordMutation(r, session.User.ID, "certificate.renew", "certificate", r.PathValue("certificateId"), err)
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
	h.recordMutation(r, session.User.ID, "certificate.redirect", "certificate", r.PathValue("certificateId"), err)
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
		writeError(w, http.StatusNotFound, "not_found", "Certificate or domain not found")
	case errors.Is(err, accounts.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "Account access is required")
	case errors.Is(err, domains.ErrDomainInactive):
		writeError(w, http.StatusConflict, "domain_inactive", "The domain must be active")
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
	if value.DomainID != "" {
		response.DomainId = &value.DomainID
	}
	return response
}

func writeCertificateJob(w http.ResponseWriter, value certificates.Certificate, job jobs.Job) {
	httpx.WriteJSON(w, http.StatusAccepted, publicapi.CertificateJobResponse{Certificate: certificateResponse(value), Job: jobResponse(job)})
}
