package httpapi

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/GVALFER/WEBYCP/internal/auth"
	publicapi "github.com/GVALFER/WEBYCP/internal/httpapi/spec"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/packages"
)

func (h *handler) listPackages(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	query, ok := requestPage(w, r)
	if !ok {
		return
	}
	page, err := h.options.Packages.Page(r.Context(), query)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response := publicapi.PackageListResponse{
		Items:      make([]publicapi.Package, 0, len(page.Items)),
		Pagination: paginationResponse(page.Query, page.Total),
	}
	for _, item := range page.Items {
		response.Items = append(response.Items, packageResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) createPackage(w http.ResponseWriter, r *http.Request, session auth.Session) {
	if session.User.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "Administrator access is required")
		return
	}
	value, ok := decodePackage(w, r)
	if !ok {
		return
	}
	created, err := h.options.Packages.Create(r.Context(), value)
	h.recordMutation(r, session.User.ID, "package.create", "package", created.ID, err)
	if err != nil {
		h.writePackageError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, packageResponse(created))
}

func (h *handler) updatePackage(w http.ResponseWriter, r *http.Request, session auth.Session) {
	if session.User.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "Administrator access is required")
		return
	}
	value, ok := decodePackage(w, r)
	if !ok {
		return
	}
	id := r.PathValue("packageId")
	updated, err := h.options.Packages.Update(r.Context(), id, value)
	h.recordMutation(r, session.User.ID, "package.update", "package", id, err)
	if err != nil {
		h.writePackageError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, packageResponse(updated))
}

func (h *handler) deletePackage(w http.ResponseWriter, r *http.Request, session auth.Session) {
	if session.User.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "Administrator access is required")
		return
	}
	id := r.PathValue("packageId")
	err := h.options.Packages.Delete(r.Context(), id)
	h.recordMutation(r, session.User.ID, "package.delete", "package", id, err)
	if err != nil {
		h.writePackageError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) assignAccountPackage(w http.ResponseWriter, r *http.Request, session auth.Session) {
	if session.User.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "Administrator access is required")
		return
	}
	var request publicapi.AssignAccountPackageRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	id := r.PathValue("accountId")
	value, err := h.options.Accounts.AssignPackage(r.Context(), id, request.PackageId, session.User.ID, true)
	h.recordMutation(r, session.User.ID, "account.package.update", "account", id, err)
	if err != nil {
		h.writePackageError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, accountOverviewResponse(value))
}

func decodePackage(w http.ResponseWriter, r *http.Request) (packages.Package, bool) {
	var request publicapi.WritePackageRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return packages.Package{}, false
	}
	return packages.Package{
		Name: request.Name,
		Limits: packages.Limits{
			Websites: request.Limits.Websites, Domains: request.Limits.Domains,
			Aliases: request.Limits.Aliases, Databases: request.Limits.Databases,
			DatabaseUsers:   request.Limits.DatabaseUsers,
			ScheduledTasks:  request.Limits.ScheduledTasks,
			BackupPlans:     request.Limits.BackupPlans,
			BackupRetention: request.Limits.BackupRetention,
			FTPAccounts:     request.Limits.FtpAccounts,
		},
	}, true
}

func packageResponse(value packages.Package) publicapi.Package {
	return publicapi.Package{
		Id: value.ID, Name: value.Name,
		Limits: publicapi.PackageLimits{
			Websites: value.Limits.Websites, Domains: value.Limits.Domains,
			Aliases: value.Limits.Aliases, Databases: value.Limits.Databases,
			DatabaseUsers:   value.Limits.DatabaseUsers,
			ScheduledTasks:  value.Limits.ScheduledTasks,
			BackupPlans:     value.Limits.BackupPlans,
			BackupRetention: value.Limits.BackupRetention,
			FtpAccounts:     value.Limits.FTPAccounts,
		},
		AccountCount: value.AccountCount,
		CreatedAt:    value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func (h *handler) writePackageError(w http.ResponseWriter, r *http.Request, err error) {
	if writeValidationError(w, err) {
		return
	}
	if writeLimitError(w, err) {
		return
	}
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, "not_found", "Package or account not found")
	case errors.Is(err, packages.ErrNameExists):
		writeError(w, http.StatusConflict, "package_name_exists", "A Package with this name already exists")
	case errors.Is(err, packages.ErrInUse):
		writeError(w, http.StatusConflict, "package_in_use", "Assign the Accounts to another Package first")
	default:
		h.internalError(w, r, err)
	}
}

func writeLimitError(w http.ResponseWriter, err error) bool {
	var limit *packages.LimitError
	if !errors.As(err, &limit) {
		return false
	}
	writeError(w, http.StatusConflict, "package_limit_reached", limit.Error())
	return true
}
