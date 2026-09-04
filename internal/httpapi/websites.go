package httpapi

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/audit"
	"github.com/GVALFER/WEBYCP/internal/auth"
	publicapi "github.com/GVALFER/WEBYCP/internal/httpapi/spec"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/websites"
)

func (h *handler) listWebsites(w http.ResponseWriter, r *http.Request, session auth.Session) {
	query, ok := requestPage(w, r)
	if !ok {
		return
	}
	page, err := h.options.Websites.WebsitePage(r.Context(), session.User.ID, session.User.Role == "admin", query)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response := publicapi.WebsiteListResponse{Items: make([]publicapi.Website, 0, len(page.Items)), Pagination: paginationResponse(page.Query, page.Total)}
	for _, item := range page.Items {
		response.Items = append(response.Items, websiteResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) createWebsite(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.CreateWebsiteRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	value := websites.Website{AccountID: request.AccountId, Name: request.Name, Kind: string(request.Kind), WebDriver: string(request.WebDriver), RuntimeDriver: string(request.RuntimeDriver), RuntimeVersion: string(request.RuntimeVersion)}
	website, _, job, err := h.options.Websites.Create(r.Context(), value, request.PrimaryDomain, session.User.ID, session.User.Role == "admin")
	if err != nil {
		h.record(r, audit.Event{UserID: session.User.ID, Action: "website.create", ResourceType: "website", Result: "failure"})
		h.writeWebsiteError(w, r, err)
		return
	}
	h.record(r, audit.Event{UserID: session.User.ID, Action: "website.create", ResourceType: "website", ResourceID: website.ID, Result: "success"})
	writeWebsiteJob(w, website, job)
}

func (h *handler) setWebsite(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.UpdateWebsiteRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	action := "website.disable"
	if request.Enabled {
		action = "website.enable"
	}
	website, job, err := h.options.Websites.SetWebsite(r.Context(), r.PathValue("websiteId"), session.User.ID, session.User.Role == "admin", request.Enabled)
	if err != nil {
		h.record(r, audit.Event{UserID: session.User.ID, Action: action, ResourceType: "website", ResourceID: r.PathValue("websiteId"), Result: "failure"})
		h.writeWebsiteError(w, r, err)
		return
	}
	h.record(r, audit.Event{UserID: session.User.ID, Action: action, ResourceType: "website", ResourceID: website.ID, Result: "success"})
	writeWebsiteJob(w, website, job)
}

func (h *handler) deleteWebsite(w http.ResponseWriter, r *http.Request, session auth.Session) {
	website, job, err := h.options.Websites.DeleteWebsite(r.Context(), r.PathValue("websiteId"), session.User.ID, session.User.Role == "admin")
	if err != nil {
		h.record(r, audit.Event{UserID: session.User.ID, Action: "website.delete", ResourceType: "website", ResourceID: r.PathValue("websiteId"), Result: "failure"})
		h.writeWebsiteError(w, r, err)
		return
	}
	h.record(r, audit.Event{UserID: session.User.ID, Action: "website.delete", ResourceType: "website", ResourceID: website.ID, Result: "success"})
	writeWebsiteJob(w, website, job)
}

func (h *handler) listWebsiteDomains(w http.ResponseWriter, r *http.Request, session auth.Session) {
	query, ok := requestPage(w, r)
	if !ok {
		return
	}
	page, err := h.options.Websites.DomainPage(r.Context(), session.User.ID, session.User.Role == "admin", r.URL.Query().Get("kind"), query)
	if err != nil {
		h.writeWebsiteError(w, r, err)
		return
	}
	response := publicapi.WebsiteDomainListResponse{Items: make([]publicapi.WebsiteDomain, 0, len(page.Items)), Pagination: paginationResponse(page.Query, page.Total)}
	for _, item := range page.Items {
		response.Items = append(response.Items, websiteDomainResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) listWebsiteDomainsForWebsite(w http.ResponseWriter, r *http.Request, session auth.Session) {
	items, err := h.options.Websites.Domains(r.Context(), r.PathValue("websiteId"), session.User.ID, session.User.Role == "admin")
	if err != nil {
		h.writeWebsiteError(w, r, err)
		return
	}
	response := publicapi.WebsiteDomainCollection{Items: make([]publicapi.WebsiteDomain, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, websiteDomainResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) createWebsiteDomain(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.CreateWebsiteDomainRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	domain, job, err := h.options.Websites.CreateDomain(r.Context(), r.PathValue("websiteId"), request.Hostname, session.User.ID, session.User.Role == "admin")
	if err != nil {
		h.record(r, audit.Event{UserID: session.User.ID, Action: "website_domain.create", ResourceType: "website_domain", Result: "failure"})
		h.writeWebsiteError(w, r, err)
		return
	}
	h.record(r, audit.Event{UserID: session.User.ID, Action: "website_domain.create", ResourceType: "website_domain", ResourceID: domain.ID, Result: "success"})
	writeWebsiteDomainJob(w, domain, job)
}

func (h *handler) setWebsiteDomain(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.UpdateWebsiteDomainRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	if (request.Enabled == nil) == (request.Hostname == nil) {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "Change either hostname or enabled")
		return
	}
	var domain websites.WebsiteDomain
	var job jobs.Job
	var err error
	action := "website_domain.update"
	if request.Hostname != nil {
		domain, job, err = h.options.Websites.RenameDomain(r.Context(), r.PathValue("websiteDomainId"), *request.Hostname, session.User.ID, session.User.Role == "admin")
	} else {
		domain, job, err = h.options.Websites.SetDomain(r.Context(), r.PathValue("websiteDomainId"), session.User.ID, session.User.Role == "admin", *request.Enabled)
		action = "website_domain.disable"
		if *request.Enabled {
			action = "website_domain.enable"
		}
	}
	if err != nil {
		h.record(r, audit.Event{UserID: session.User.ID, Action: action, ResourceType: "website_domain", ResourceID: r.PathValue("websiteDomainId"), Result: "failure"})
		h.writeWebsiteError(w, r, err)
		return
	}
	h.record(r, audit.Event{UserID: session.User.ID, Action: action, ResourceType: "website_domain", ResourceID: domain.ID, Result: "success"})
	writeWebsiteDomainJob(w, domain, job)
}

func (h *handler) deleteWebsiteDomain(w http.ResponseWriter, r *http.Request, session auth.Session) {
	domain, job, err := h.options.Websites.DeleteDomain(r.Context(), r.PathValue("websiteDomainId"), session.User.ID, session.User.Role == "admin")
	if err != nil {
		h.record(r, audit.Event{UserID: session.User.ID, Action: "website_domain.delete", ResourceType: "website_domain", ResourceID: r.PathValue("websiteDomainId"), Result: "failure"})
		h.writeWebsiteError(w, r, err)
		return
	}
	h.record(r, audit.Event{UserID: session.User.ID, Action: "website_domain.delete", ResourceType: "website_domain", ResourceID: domain.ID, Result: "success"})
	writeWebsiteDomainJob(w, domain, job)
}

func websiteResponse(value websites.Website) publicapi.Website {
	return publicapi.Website{Id: value.ID, AccountId: value.AccountID, NodeId: value.NodeID, Name: value.Name, Kind: publicapi.WebsiteKind(value.Kind), DocumentRoot: value.DocumentRoot, WebDriver: publicapi.WebsiteWebDriver(value.WebDriver), RuntimeDriver: publicapi.WebsiteRuntimeDriver(value.RuntimeDriver), RuntimeVersion: publicapi.WebsiteRuntimeVersion(value.RuntimeVersion), Status: publicapi.WebsiteStatus(value.Status), Enabled: value.Enabled, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func websiteDomainResponse(value websites.WebsiteDomain) publicapi.WebsiteDomain {
	return publicapi.WebsiteDomain{Id: value.ID, WebsiteId: value.WebsiteID, Hostname: value.Hostname, Kind: publicapi.WebsiteDomainKind(value.Kind), Status: publicapi.WebsiteDomainStatus(value.Status), Enabled: value.Enabled, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func (h *handler) writeWebsiteError(w http.ResponseWriter, r *http.Request, err error) {
	if writeValidationError(w, err) {
		return
	}
	if writeLimitError(w, err) {
		return
	}
	switch {
	case errors.Is(err, accounts.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "Account access is required")
	case errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, "not_found", "Website, domain or account not found")
	case errors.Is(err, accounts.ErrBusy), errors.Is(err, websites.ErrWebsiteInactive):
		writeError(w, http.StatusConflict, "resource_inactive", "The account and website must be active")
	case errors.Is(err, websites.ErrWebsiteBusy), errors.Is(err, websites.ErrWebsiteDomainBusy):
		writeError(w, http.StatusConflict, "resource_busy", "A resource operation is already pending")
	case errors.Is(err, websites.ErrHostnameExists):
		writeError(w, http.StatusConflict, "hostname_exists", "This hostname is already in use")
	case errors.Is(err, websites.ErrHostnameSame):
		writeError(w, http.StatusConflict, "hostname_unchanged", "The hostname is unchanged")
	case errors.Is(err, websites.ErrPrimaryRequired):
		writeError(w, http.StatusConflict, "primary_domain_required", "The primary domain cannot be disabled or deleted")
	default:
		h.internalError(w, r, err)
	}
}

func writeWebsiteJob(w http.ResponseWriter, website websites.Website, job jobs.Job) {
	httpx.WriteJSON(w, http.StatusAccepted, publicapi.WebsiteJobResponse{Website: websiteResponse(website), Job: jobResponse(job)})
}

func writeWebsiteDomainJob(w http.ResponseWriter, domain websites.WebsiteDomain, job jobs.Job) {
	httpx.WriteJSON(w, http.StatusAccepted, publicapi.WebsiteDomainJobResponse{Domain: websiteDomainResponse(domain), Job: jobResponse(job)})
}
