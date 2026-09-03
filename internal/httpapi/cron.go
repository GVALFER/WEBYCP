package httpapi

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/auth"
	cronjob "github.com/GVALFER/WEBYCP/internal/cron"
	"github.com/GVALFER/WEBYCP/internal/httpapi/spec"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/jobs"
)

func (h *handler) listCronJobs(w http.ResponseWriter, r *http.Request, session auth.Session) {
	items, err := h.options.Cron.CronJobs(r.Context(), session.User.ID, session.User.Role == "admin")
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response := publicapi.CronJobListResponse{Items: make([]publicapi.CronJob, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, cronJobResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) createCronJob(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.WriteCronJobRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	value, job, err := h.options.Cron.Create(r.Context(), request.AccountId, request.Name, request.Schedule, request.Command, session.User.ID, session.User.Role == "admin", request.Enabled)
	h.recordMutation(r, session.User.ID, "cron.create", "cron_job", value.ID, err)
	if err != nil {
		h.writeCronError(w, r, err)
		return
	}
	writeCronJob(w, value, job)
}

func (h *handler) setCronJob(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.WriteCronJobRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	value, job, err := h.options.Cron.Update(r.Context(), r.PathValue("cronJobId"), request.AccountId, request.Name, request.Schedule, request.Command, session.User.ID, session.User.Role == "admin", request.Enabled)
	h.recordMutation(r, session.User.ID, "cron.update", "cron_job", r.PathValue("cronJobId"), err)
	if err != nil {
		h.writeCronError(w, r, err)
		return
	}
	writeCronJob(w, value, job)
}

func (h *handler) deleteCronJob(w http.ResponseWriter, r *http.Request, session auth.Session) {
	job, err := h.options.Cron.Delete(r.Context(), r.PathValue("cronJobId"), session.User.ID, session.User.Role == "admin")
	h.recordMutation(r, session.User.ID, "cron.delete", "cron_job", r.PathValue("cronJobId"), err)
	if err != nil {
		h.writeCronError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, jobResponse(job))
}

func (h *handler) writeCronError(w http.ResponseWriter, r *http.Request, err error) {
	if writeValidationError(w, err) {
		return
	}
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, "not_found", "Cron job not found")
	case errors.Is(err, accounts.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "Account access is required")
	case errors.Is(err, accounts.ErrBusy):
		writeError(w, http.StatusConflict, "account_inactive", "The account must be active")
	default:
		h.internalError(w, r, err)
	}
}

func cronJobResponse(value cronjob.CronJob) publicapi.CronJob {
	return publicapi.CronJob{Id: value.ID, AccountId: value.AccountID, NodeId: value.NodeID, Name: value.Name, Schedule: value.Schedule, Command: value.Command, Enabled: value.Enabled, Status: publicapi.CronJobStatus(value.Status), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func writeCronJob(w http.ResponseWriter, value cronjob.CronJob, job jobs.Job) {
	httpx.WriteJSON(w, http.StatusAccepted, publicapi.CronJobResponse{CronJob: cronJobResponse(value), Job: jobResponse(job)})
}
