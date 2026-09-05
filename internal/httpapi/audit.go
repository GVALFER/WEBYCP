package httpapi

import (
	"net/http"

	"github.com/GVALFER/WEBYCP/internal/audit"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/httpapi/spec"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

func (h *handler) listAuditEvents(w http.ResponseWriter, r *http.Request, session auth.Session) {
	if session.User.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "Administrator access is required")
		return
	}
	query, ok := requestPage(w, r)
	if !ok {
		return
	}
	jobID := r.URL.Query().Get("jobId")
	if jobID != "" {
		if err := validate.ID("jobId", jobID); err != nil {
			writeValidationError(w, err)
			return
		}
	}
	page, err := h.options.Audit.AuditPage(r.Context(), query, jobID)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	items := make([]publicapi.AuditEvent, 0, len(page.Items))
	for _, event := range page.Items {
		item := publicapi.AuditEvent{Id: event.ID, Action: event.Action, ResourceType: event.ResourceType, Result: event.Result, CreatedAt: event.CreatedAt}
		if event.UserID != "" {
			item.UserId = &event.UserID
		}
		if event.ResourceID != "" {
			item.ResourceId = &event.ResourceID
		}
		if event.JobID != "" {
			item.JobId = &event.JobID
		}
		items = append(items, item)
	}
	httpx.WriteJSON(w, http.StatusOK, publicapi.AuditEventListResponse{
		Items: items, Pagination: paginationResponse(page.Query, page.Total),
	})
}

func (h *handler) recordJobMutation(r *http.Request, userID, action, resourceType, resourceID, jobID string, err error) {
	result := "success"
	if err != nil {
		result = "failure"
	}
	h.record(r, audit.Event{
		UserID: userID, Action: action, ResourceType: resourceType,
		ResourceID: resourceID, JobID: jobID, Result: result,
	})
}
