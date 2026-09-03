package httpapi

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/backupfmt"
	"github.com/GVALFER/WEBYCP/internal/backups"
	"github.com/GVALFER/WEBYCP/internal/httpapi/spec"
	"github.com/GVALFER/WEBYCP/internal/httpx"
)

func (h *handler) listBackupPlans(w http.ResponseWriter, r *http.Request, session auth.Session) {
	items, err := h.options.Backups.Plans(r.Context(), session.User.ID, session.User.Role == "admin")
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response := publicapi.BackupPlanListResponse{Items: make([]publicapi.BackupPlan, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, backupPlanResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) createBackupPlan(w http.ResponseWriter, r *http.Request, session auth.Session) {
	request, ok := decodeBackupPlan(w, r)
	if !ok {
		return
	}
	value, err := h.options.Backups.CreatePlan(r.Context(), request, session.User.ID, session.User.Role == "admin")
	h.recordMutation(r, session.User.ID, "backup_plan.create", "backup_plan", value.ID, err)
	if err != nil {
		h.writeBackupError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, backupPlanResponse(value))
}

func (h *handler) setBackupPlan(w http.ResponseWriter, r *http.Request, session auth.Session) {
	request, ok := decodeBackupPlan(w, r)
	if !ok {
		return
	}
	value, err := h.options.Backups.UpdatePlan(r.Context(), r.PathValue("backupPlanId"), request, session.User.ID, session.User.Role == "admin")
	h.recordMutation(r, session.User.ID, "backup_plan.update", "backup_plan", r.PathValue("backupPlanId"), err)
	if err != nil {
		h.writeBackupError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, backupPlanResponse(value))
}

func decodeBackupPlan(w http.ResponseWriter, r *http.Request) (backups.Plan, bool) {
	var request publicapi.WriteBackupPlanRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return backups.Plan{}, false
	}
	return backups.Plan{AccountID: request.AccountId, Name: request.Name, Schedule: request.Schedule, RetentionCount: int64(request.RetentionCount), IncludeFiles: request.IncludeFiles, IncludeDatabases: request.IncludeDatabases, Enabled: request.Enabled}, true
}

func (h *handler) deleteBackupPlan(w http.ResponseWriter, r *http.Request, session auth.Session) {
	err := h.options.Backups.DeletePlan(r.Context(), r.PathValue("backupPlanId"), session.User.ID, session.User.Role == "admin")
	h.recordMutation(r, session.User.ID, "backup_plan.delete", "backup_plan", r.PathValue("backupPlanId"), err)
	if err != nil {
		h.writeBackupError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) runBackupPlan(w http.ResponseWriter, r *http.Request, session auth.Session) {
	run, job, err := h.options.Backups.Run(r.Context(), r.PathValue("backupPlanId"), session.User.ID, session.User.Role == "admin")
	h.recordMutation(r, session.User.ID, "backup.run", "backup_plan", r.PathValue("backupPlanId"), err)
	if err != nil {
		h.writeBackupError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, publicapi.BackupRunResponse{Run: backupRunResponse(run), Job: jobResponse(job)})
}

func (h *handler) listBackupRuns(w http.ResponseWriter, r *http.Request, session auth.Session) {
	items, err := h.options.Backups.Runs(r.Context(), session.User.ID, session.User.Role == "admin")
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response := publicapi.BackupRunListResponse{Items: make([]publicapi.BackupRun, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, backupRunResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) listBackupArtifacts(w http.ResponseWriter, r *http.Request, session auth.Session) {
	items, err := h.options.Backups.Artifacts(r.Context(), session.User.ID, session.User.Role == "admin")
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response := publicapi.BackupArtifactListResponse{Items: make([]publicapi.BackupArtifact, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, backupArtifactResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) deleteBackupArtifact(w http.ResponseWriter, r *http.Request, session auth.Session) {
	err := h.options.Backups.DeleteArtifact(
		r.Context(), r.PathValue("backupArtifactId"), session.User.ID,
		session.User.Role == "admin",
	)
	h.recordMutation(r, session.User.ID, "backup_artifact.delete", "backup_artifact", r.PathValue("backupArtifactId"), err)
	if err != nil {
		h.writeBackupError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) previewBackupRestore(w http.ResponseWriter, r *http.Request, session auth.Session) {
	manifest, err := h.options.Backups.Preview(r.Context(), r.PathValue("backupArtifactId"), session.User.ID, session.User.Role == "admin")
	if err != nil {
		h.writeBackupError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, backupManifestResponse(manifest))
}

func (h *handler) restoreBackup(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.RestoreBackupRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	job, err := h.options.Backups.Restore(r.Context(), r.PathValue("backupArtifactId"), session.User.ID, session.User.Role == "admin", backups.RestoreScope{Files: request.Files, Databases: request.Databases, Metadata: request.Metadata})
	h.recordMutation(r, session.User.ID, "backup.restore", "backup_artifact", r.PathValue("backupArtifactId"), err)
	if err != nil {
		h.writeBackupError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, jobResponse(job))
}

func (h *handler) writeBackupError(w http.ResponseWriter, r *http.Request, err error) {
	if writeValidationError(w, err) {
		return
	}
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, "not_found", "Backup resource not found")
	case errors.Is(err, accounts.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "Account access is required")
	case errors.Is(err, accounts.ErrBusy):
		writeError(w, http.StatusConflict, "account_inactive", "The account must be active")
	case errors.Is(err, backups.ErrBusy):
		writeError(w, http.StatusConflict, "backup_busy", "The backup plan already has an active run")
	case errors.Is(err, backups.ErrScope):
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "Select at least one backup scope")
	default:
		h.internalError(w, r, err)
	}
}

func backupPlanResponse(value backups.Plan) publicapi.BackupPlan {
	return publicapi.BackupPlan{Id: value.ID, AccountId: value.AccountID, NodeId: value.NodeID, Name: value.Name, Schedule: value.Schedule, RetentionCount: int(value.RetentionCount), IncludeFiles: value.IncludeFiles, IncludeDatabases: value.IncludeDatabases, Enabled: value.Enabled, LastRunAt: value.LastRunAt, NextRunAt: value.NextRunAt, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func backupRunResponse(value backups.Run) publicapi.BackupRun {
	return publicapi.BackupRun{Id: value.ID, PlanId: value.PlanID, AccountId: value.AccountID, NodeId: value.NodeID, Status: publicapi.BackupRunStatus(value.Status), Error: value.Error, CreatedAt: value.CreatedAt, StartedAt: value.StartedAt, FinishedAt: value.FinishedAt}
}

func backupArtifactResponse(value backups.Artifact) publicapi.BackupArtifact {
	return publicapi.BackupArtifact{Id: value.ID, RunId: value.RunID, AccountId: value.AccountID, NodeId: value.NodeID, Checksum: value.Checksum, Size: value.Size, Manifest: backupManifestResponse(value.Manifest), CreatedAt: value.CreatedAt}
}

func backupManifestResponse(value backupfmt.Manifest) publicapi.BackupManifest {
	entries := make([]publicapi.BackupEntry, 0, len(value.Entries))
	for _, entry := range value.Entries {
		entries = append(entries, publicapi.BackupEntry{Path: entry.Path, Size: entry.Size, Checksum: entry.Checksum})
	}
	return publicapi.BackupManifest{Version: value.Version, RunId: value.RunID, AccountId: value.AccountID, CreatedAt: value.CreatedAt, Files: value.Files, Databases: value.Databases, Metadata: value.Metadata, Entries: entries}
}
