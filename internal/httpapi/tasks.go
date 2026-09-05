package httpapi

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/httpapi/spec"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/tasks"
)

func (h *handler) listScheduledTasks(w http.ResponseWriter, r *http.Request, session auth.Session) {
	query, ok := requestPage(w, r)
	if !ok {
		return
	}
	page, err := h.options.Tasks.ScheduledTaskPage(
		r.Context(), session.User.ID, session.User.Role == "admin", query,
	)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response := publicapi.ScheduledTaskListResponse{
		Items:      make([]publicapi.ScheduledTask, 0, len(page.Items)),
		Pagination: paginationResponse(page.Query, page.Total),
	}
	for _, item := range page.Items {
		response.Items = append(response.Items, scheduledTaskResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) createScheduledTask(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.WriteScheduledTaskRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	value, job, err := h.options.Tasks.Create(r.Context(), request.AccountId, request.Name, request.Schedule, request.Command, string(request.SchedulerDriver), session.User.ID, tasks.Kind(request.Kind), session.User.Role == "admin", request.Enabled)
	h.recordJobMutation(r, session.User.ID, "task.create", "scheduled_task", value.ID, job.ID, err)
	if err != nil {
		h.writeTaskError(w, r, err)
		return
	}
	writeScheduledTask(w, value, job)
}

func (h *handler) setScheduledTask(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.WriteScheduledTaskRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	value, job, err := h.options.Tasks.Update(r.Context(), r.PathValue("scheduledTaskId"), request.AccountId, request.Name, request.Schedule, request.Command, string(request.SchedulerDriver), session.User.ID, tasks.Kind(request.Kind), session.User.Role == "admin", request.Enabled)
	h.recordJobMutation(r, session.User.ID, "task.update", "scheduled_task", r.PathValue("scheduledTaskId"), job.ID, err)
	if err != nil {
		h.writeTaskError(w, r, err)
		return
	}
	writeScheduledTask(w, value, job)
}

func (h *handler) deleteScheduledTask(w http.ResponseWriter, r *http.Request, session auth.Session) {
	job, err := h.options.Tasks.Delete(r.Context(), r.PathValue("scheduledTaskId"), session.User.ID, session.User.Role == "admin")
	h.recordJobMutation(r, session.User.ID, "task.delete", "scheduled_task", r.PathValue("scheduledTaskId"), job.ID, err)
	if err != nil {
		h.writeTaskError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, jobResponse(job))
}

func (h *handler) writeTaskError(w http.ResponseWriter, r *http.Request, err error) {
	if writeValidationError(w, err) {
		return
	}
	if writeLimitError(w, err) {
		return
	}
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, "not_found", "Scheduled task not found")
	case errors.Is(err, accounts.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "Account access is required")
	case errors.Is(err, accounts.ErrBusy):
		writeError(w, http.StatusConflict, "account_inactive", "The account must be active")
	default:
		h.internalError(w, r, err)
	}
}

func scheduledTaskResponse(value tasks.ScheduledTask) publicapi.ScheduledTask {
	return publicapi.ScheduledTask{Kind: publicapi.TaskKind(value.Kind), Id: value.ID, AccountId: value.AccountID, NodeId: value.NodeID, Name: value.Name, Schedule: value.Schedule, Command: value.Command, SchedulerDriver: publicapi.ScheduledTaskSchedulerDriver(value.SchedulerDriver), Enabled: value.Enabled, Status: publicapi.ScheduledTaskStatus(value.Status), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func writeScheduledTask(w http.ResponseWriter, value tasks.ScheduledTask, job jobs.Job) {
	httpx.WriteJSON(w, http.StatusAccepted, publicapi.ScheduledTaskResponse{ScheduledTask: scheduledTaskResponse(value), Job: jobResponse(job)})
}
