package httpapi

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/databases"
	"github.com/GVALFER/WEBYCP/internal/httpapi/spec"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/jobs"
)

func (h *handler) listDatabases(w http.ResponseWriter, r *http.Request, session auth.Session) {
	query, ok := requestPage(w, r)
	if !ok {
		return
	}
	page, err := h.options.Databases.DatabasePage(
		r.Context(), session.User.ID, session.User.Role == "admin", query,
	)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response := publicapi.DatabaseListResponse{
		Items:      make([]publicapi.Database, 0, len(page.Items)),
		Pagination: paginationResponse(page.Query, page.Total),
	}
	for _, item := range page.Items {
		response.Items = append(response.Items, databaseResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) createDatabase(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.CreateDatabaseRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	value, job, err := h.options.Databases.CreateDatabase(r.Context(), request.AccountId, request.Name, string(request.Driver), session.User.ID, session.User.Role == "admin")
	h.recordMutation(r, session.User.ID, "database.create", "database", value.ID, err)
	if err != nil {
		h.writeDatabaseError(w, r, err)
		return
	}
	writeDatabaseJob(w, value, job)
}

func (h *handler) deleteDatabase(w http.ResponseWriter, r *http.Request, session auth.Session) {
	value, job, err := h.options.Databases.DeleteDatabase(r.Context(), r.PathValue("databaseId"), session.User.ID, session.User.Role == "admin")
	h.recordMutation(r, session.User.ID, "database.delete", "database", r.PathValue("databaseId"), err)
	if err != nil {
		h.writeDatabaseError(w, r, err)
		return
	}
	writeDatabaseJob(w, value, job)
}

func (h *handler) listDatabaseUsers(w http.ResponseWriter, r *http.Request, session auth.Session) {
	query, ok := requestPage(w, r)
	if !ok {
		return
	}
	page, err := h.options.Databases.UserPage(
		r.Context(), session.User.ID, session.User.Role == "admin", query,
	)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response := publicapi.DatabaseUserListResponse{
		Items:      make([]publicapi.DatabaseUser, 0, len(page.Items)),
		Pagination: paginationResponse(page.Query, page.Total),
	}
	for _, item := range page.Items {
		response.Items = append(response.Items, databaseUserResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) createDatabaseUser(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.CreateDatabaseUserRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	value, job, password, err := h.options.Databases.CreateUser(r.Context(), request.AccountId, request.Name, string(request.Driver), session.User.ID, session.User.Role == "admin")
	h.recordMutation(r, session.User.ID, "database_user.create", "database_user", value.ID, err)
	if err != nil {
		h.writeDatabaseError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, publicapi.DatabaseUserJobResponse{DatabaseUser: databaseUserResponse(value), Job: jobResponse(job), Password: &password})
}

func (h *handler) deleteDatabaseUser(w http.ResponseWriter, r *http.Request, session auth.Session) {
	value, job, err := h.options.Databases.DeleteUser(r.Context(), r.PathValue("databaseUserId"), session.User.ID, session.User.Role == "admin")
	h.recordMutation(r, session.User.ID, "database_user.delete", "database_user", r.PathValue("databaseUserId"), err)
	if err != nil {
		h.writeDatabaseError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, publicapi.DatabaseUserJobResponse{DatabaseUser: databaseUserResponse(value), Job: jobResponse(job)})
}

func (h *handler) listDatabaseGrants(w http.ResponseWriter, r *http.Request, session auth.Session) {
	query, ok := requestPage(w, r)
	if !ok {
		return
	}
	page, err := h.options.Databases.GrantPage(
		r.Context(), session.User.ID, session.User.Role == "admin", query,
	)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response := publicapi.DatabaseGrantListResponse{
		Items:      make([]publicapi.DatabaseGrant, 0, len(page.Items)),
		Pagination: paginationResponse(page.Query, page.Total),
	}
	for _, item := range page.Items {
		response.Items = append(response.Items, databaseGrantResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) createDatabaseGrant(w http.ResponseWriter, r *http.Request, session auth.Session) {
	h.setDatabaseGrant(w, r, session, true)
}

func (h *handler) deleteDatabaseGrant(w http.ResponseWriter, r *http.Request, session auth.Session) {
	h.setDatabaseGrant(w, r, session, false)
}

func (h *handler) setDatabaseGrant(w http.ResponseWriter, r *http.Request, session auth.Session, enabled bool) {
	grant, job, err := h.options.Databases.SetGrant(r.Context(), r.PathValue("databaseId"), r.PathValue("databaseUserId"), session.User.ID, session.User.Role == "admin", enabled)
	action := "database_grant.delete"
	if enabled {
		action = "database_grant.create"
	}
	h.recordMutation(r, session.User.ID, action, "database_grant", r.PathValue("databaseId")+":"+r.PathValue("databaseUserId"), err)
	if err != nil {
		h.writeDatabaseError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, publicapi.DatabaseGrantJobResponse{Grant: databaseGrantResponse(grant), Job: jobResponse(job)})
}

func (h *handler) writeDatabaseError(w http.ResponseWriter, r *http.Request, err error) {
	if writeValidationError(w, err) {
		return
	}
	if writeLimitError(w, err) {
		return
	}
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, "not_found", "Database resource not found")
	case errors.Is(err, accounts.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "Account access is required")
	case errors.Is(err, databases.ErrNameExists):
		writeError(w, http.StatusConflict, "name_exists", "This database resource name is already in use")
	case errors.Is(err, databases.ErrBusy), errors.Is(err, accounts.ErrBusy):
		writeError(w, http.StatusConflict, "resource_busy", "The database resource is not ready")
	case errors.Is(err, databases.ErrCrossAccount):
		writeError(w, http.StatusConflict, "cross_account_grant", "Database grants must stay within one account")
	default:
		h.internalError(w, r, err)
	}
}

func databaseResponse(value databases.Database) publicapi.Database {
	return publicapi.Database{Id: value.ID, AccountId: value.AccountID, NodeId: value.NodeID, Name: value.Name, SystemName: value.SystemName, Driver: publicapi.DatabaseDriver(value.Driver), Status: publicapi.DatabaseStatus(value.Status), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func databaseUserResponse(value databases.User) publicapi.DatabaseUser {
	return publicapi.DatabaseUser{Id: value.ID, AccountId: value.AccountID, NodeId: value.NodeID, Name: value.Name, SystemName: value.SystemName, Driver: publicapi.DatabaseUserDriver(value.Driver), Status: publicapi.DatabaseUserStatus(value.Status), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func databaseGrantResponse(value databases.Grant) publicapi.DatabaseGrant {
	return publicapi.DatabaseGrant{DatabaseId: value.DatabaseID, DatabaseUserId: value.UserID, Status: publicapi.DatabaseGrantStatus(value.Status), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func writeDatabaseJob(w http.ResponseWriter, value databases.Database, job jobs.Job) {
	httpx.WriteJSON(w, http.StatusAccepted, publicapi.DatabaseJobResponse{Database: databaseResponse(value), Job: jobResponse(job)})
}
