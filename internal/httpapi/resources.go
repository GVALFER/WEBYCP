package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/audit"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/httpapi/spec"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/nodes"
	"github.com/GVALFER/WEBYCP/internal/services"
)

func (h *handler) listAccounts(w http.ResponseWriter, r *http.Request, session auth.Session) {
	query, ok := requestPage(w, r)
	if !ok {
		return
	}
	page, err := h.options.Accounts.AccountPage(
		r.Context(), session.User.ID, session.User.Role == "admin", query,
	)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response := publicapi.AccountListResponse{
		Items:      make([]publicapi.AccountOverview, 0, len(page.Items)),
		Pagination: paginationResponse(page.Query, page.Total),
	}
	for _, item := range page.Items {
		response.Items = append(response.Items, accountOverviewResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) createAccount(w http.ResponseWriter, r *http.Request, session auth.Session) {
	if session.User.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "Administrator access is required")
		return
	}
	var request publicapi.CreateAccountRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	account, job, err := h.options.Accounts.Create(
		r.Context(), request.Name, request.NodeId, request.PackageId, session.User.ID,
	)
	if err != nil {
		h.record(r, audit.Event{
			UserID: session.User.ID, Action: "account.create", ResourceType: "account",
			Result: "failure",
		})
		if writeValidationError(w, err) {
			return
		}
		if errors.Is(err, accounts.ErrNameExists) {
			writeError(w, http.StatusConflict, "account_name_exists", "An account with this name already exists")
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not_found", "Node or Package not found")
			return
		}
		h.internalError(w, r, err)
		return
	}
	h.record(r, audit.Event{
		UserID: session.User.ID, Action: "account.create", ResourceType: "account",
		ResourceID: account.ID, JobID: job.ID, Result: "success",
	})
	httpx.WriteJSON(w, http.StatusAccepted, publicapi.AccountJobResponse{
		Account: accountResponse(account), Job: jobResponse(job),
	})
}

func (h *handler) setAccount(w http.ResponseWriter, r *http.Request, session auth.Session) {
	if session.User.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "Administrator access is required")
		return
	}
	var request publicapi.UpdateEnabledRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	account, job, err := h.options.Accounts.Set(
		r.Context(), r.PathValue("accountId"), session.User.ID, true, request.Enabled,
	)
	if err != nil {
		action := "account.disable"
		if request.Enabled {
			action = "account.enable"
		}
		h.recordMutation(r, session.User.ID, action, "account", r.PathValue("accountId"), err)
		h.writeAccountError(w, r, err)
		return
	}
	h.record(r, audit.Event{
		UserID: session.User.ID, Action: job.Kind, ResourceType: "account",
		ResourceID: account.ID, JobID: job.ID, Result: "success",
	})
	writeAccountJob(w, account, job)
}

func (h *handler) deleteAccount(w http.ResponseWriter, r *http.Request, session auth.Session) {
	if session.User.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "Administrator access is required")
		return
	}
	account, job, err := h.options.Accounts.Delete(
		r.Context(), r.PathValue("accountId"), session.User.ID, true,
	)
	if err != nil {
		h.recordMutation(r, session.User.ID, "account.delete", "account", r.PathValue("accountId"), err)
		h.writeAccountError(w, r, err)
		return
	}
	h.record(r, audit.Event{
		UserID: session.User.ID, Action: "account.delete", ResourceType: "account",
		ResourceID: account.ID, JobID: job.ID, Result: "success",
	})
	writeAccountJob(w, account, job)
}

func (h *handler) listNodes(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	items, err := h.options.Nodes.Nodes(r.Context())
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response := publicapi.NodeListResponse{Items: make([]publicapi.Node, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, nodeResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) probeNode(w http.ResponseWriter, r *http.Request, session auth.Session) {
	if session.User.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "Administrator access is required")
		return
	}
	nodeID := r.PathValue("nodeId")
	if _, err := h.options.Nodes.Node(r.Context(), nodeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not_found", "Node not found")
			return
		}
		h.internalError(w, r, err)
		return
	}
	job, err := h.options.Jobs.QueueProbe(r.Context(), nodeID, session.User.ID)
	if err != nil {
		h.record(r, audit.Event{
			UserID: session.User.ID, Action: "node.probe", ResourceType: "node",
			ResourceID: nodeID, Result: "failure",
		})
		h.internalError(w, r, err)
		return
	}
	h.record(r, audit.Event{
		UserID: session.User.ID, Action: "node.probe", ResourceType: "node",
		ResourceID: nodeID, JobID: job.ID, Result: "success",
	})
	httpx.WriteJSON(w, http.StatusAccepted, jobResponse(job))
}

func (h *handler) listJobs(w http.ResponseWriter, r *http.Request, session auth.Session) {
	if session.User.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "Administrator access is required")
		return
	}
	query, ok := requestPage(w, r)
	if !ok {
		return
	}
	page, err := h.options.Jobs.JobPage(r.Context(), query)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response := publicapi.JobListResponse{
		Items:      make([]publicapi.Job, 0, len(page.Items)),
		Pagination: paginationResponse(page.Query, page.Total),
	}
	for _, item := range page.Items {
		response.Items = append(response.Items, jobResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) getJob(w http.ResponseWriter, r *http.Request, session auth.Session) {
	if session.User.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "Administrator access is required")
		return
	}
	job, steps, err := h.options.Jobs.Job(r.Context(), r.PathValue("jobId"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not_found", "Job not found")
			return
		}
		h.internalError(w, r, err)
		return
	}
	response := publicapi.JobDetail{
		Job: jobResponse(job), Steps: make([]publicapi.JobStep, 0, len(steps)),
	}
	for _, step := range steps {
		response.Steps = append(response.Steps, stepResponse(step))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func nodeResponse(node nodes.Node) publicapi.Node {
	response := publicapi.Node{
		Id: node.ID, Name: node.Name, Kind: publicapi.NodeKind(node.Kind),
		Endpoint: node.Endpoint, Status: publicapi.NodeStatus(node.Status),
		LastSeenAt: node.LastSeenAt, CapabilitiesAt: node.CapabilitiesAt,
		CreatedAt: node.CreatedAt, UpdatedAt: node.UpdatedAt,
	}
	if node.Capabilities != nil {
		value := serviceCapabilitiesResponse(*node.Capabilities)
		response.Capabilities = &value
	}
	return response
}

func serviceCapabilitiesResponse(value services.Capabilities) publicapi.ServiceCapabilities {
	return publicapi.ServiceCapabilities{
		Webservers: publicCapabilityValues(value.Webservers),
		Runtimes:   publicCapabilityValues(value.Runtimes),
		Databases:  publicCapabilityValues(value.Databases),
		Schedulers: publicCapabilityValues(value.Schedulers),
		Backups:    publicCapabilityValues(value.Backups),
		Dns:        publicCapabilityValues(value.DNS),
	}
}

func publicCapabilityValues(values []services.Capability) []publicapi.ServiceCapability {
	result := make([]publicapi.ServiceCapability, 0, len(values))
	for _, value := range values {
		result = append(result, publicapi.ServiceCapability{
			Driver: value.Driver, Version: value.Version,
			Status: publicapi.ServiceCapabilityStatus(value.Status),
		})
	}
	return result
}

func accountResponse(account accounts.Account) publicapi.Account {
	return publicapi.Account{
		Id: account.ID, NodeId: account.NodeID, Name: account.Name,
		SystemUser: account.SystemUser, Status: publicapi.AccountStatus(account.Status),
		Enabled:   account.Enabled,
		CreatedAt: account.CreatedAt, UpdatedAt: account.UpdatedAt,
	}
}

func accountOverviewResponse(value accounts.Overview) publicapi.AccountOverview {
	return publicapi.AccountOverview{
		Id: value.ID, NodeId: value.NodeID, Name: value.Name,
		SystemUser: value.SystemUser, Status: publicapi.AccountOverviewStatus(value.Status),
		Enabled: value.Enabled, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
		Package: packageResponse(value.Package),
		Usage: publicapi.PackageUsage{
			Websites: value.Usage.Websites, Domains: value.Usage.Domains,
			Aliases: value.Usage.Aliases, Databases: value.Usage.Databases,
			DatabaseUsers:  value.Usage.DatabaseUsers,
			ScheduledTasks: value.Usage.ScheduledTasks, BackupPlans: value.Usage.BackupPlans,
			FtpAccounts: value.Usage.FTPAccounts,
		},
	}
}

func (h *handler) writeAccountError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, "not_found", "Account not found")
	case errors.Is(err, accounts.ErrBusy):
		writeError(w, http.StatusConflict, "resource_busy", "An account operation is already pending")
	case errors.Is(err, accounts.ErrNotEmpty):
		writeError(w, http.StatusConflict, "account_not_empty", "Delete the account resources first")
	default:
		h.internalError(w, r, err)
	}
}

func writeAccountJob(w http.ResponseWriter, account accounts.Account, job jobs.Job) {
	httpx.WriteJSON(w, http.StatusAccepted, publicapi.AccountJobResponse{
		Account: accountResponse(account), Job: jobResponse(job),
	})
}

func jobResponse(job jobs.Job) publicapi.Job {
	payload := make(map[string]interface{})
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
		payload = map[string]interface{}{}
	}
	delete(payload, "password")
	response := publicapi.Job{
		Id: job.ID, Kind: job.Kind, Status: publicapi.JobStatus(job.Status), Payload: payload,
		Attempts: job.Attempts, MaxAttempts: job.MaxAttempts, CreatedAt: job.CreatedAt,
		StartedAt: job.StartedAt, FinishedAt: job.FinishedAt,
	}
	if job.NodeID != "" {
		response.NodeId = &job.NodeID
	}
	if job.UserID != "" {
		response.UserId = &job.UserID
	}
	if job.Error != "" {
		response.Error = &job.Error
	}
	return response
}

func stepResponse(step jobs.Step) publicapi.JobStep {
	return publicapi.JobStep{
		Id: step.ID, Name: step.Name, Status: publicapi.JobStepStatus(step.Status),
		Message: step.Message, StartedAt: step.StartedAt, FinishedAt: step.FinishedAt,
	}
}
