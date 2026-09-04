package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/audit"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/domains"
	"github.com/GVALFER/WEBYCP/internal/httpapi/spec"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/nodes"
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
		Items:      make([]publicapi.Account, 0, len(page.Items)),
		Pagination: paginationResponse(page.Query, page.Total),
	}
	for _, item := range page.Items {
		response.Items = append(response.Items, accountResponse(item))
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
		r.Context(), request.Name, request.NodeId, session.User.ID,
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
			writeError(w, http.StatusNotFound, "not_found", "Node not found")
			return
		}
		h.internalError(w, r, err)
		return
	}
	h.record(r, audit.Event{
		UserID: session.User.ID, Action: "account.create", ResourceType: "account",
		ResourceID: account.ID, Result: "success",
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
		ResourceID: account.ID, Result: "success",
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
		ResourceID: account.ID, Result: "success",
	})
	writeAccountJob(w, account, job)
}

func (h *handler) listDomains(w http.ResponseWriter, r *http.Request, session auth.Session) {
	query, ok := requestPage(w, r)
	if !ok {
		return
	}
	page, err := h.options.Domains.DomainPage(
		r.Context(), session.User.ID, session.User.Role == "admin", query,
	)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response := publicapi.DomainListResponse{
		Items:      make([]publicapi.Domain, 0, len(page.Items)),
		Pagination: paginationResponse(page.Query, page.Total),
	}
	for _, item := range page.Items {
		response.Items = append(response.Items, domainResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) createDomain(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.CreateDomainRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	domain, job, err := h.options.Domains.Create(
		r.Context(), request.AccountId, request.Name,
		session.User.ID, session.User.Role == "admin",
	)
	if err != nil {
		h.record(r, audit.Event{
			UserID: session.User.ID, Action: "domain.create", ResourceType: "domain",
			Result: "failure",
		})
		h.writeDomainError(w, r, err)
		return
	}
	h.record(r, audit.Event{
		UserID: session.User.ID, Action: "domain.create", ResourceType: "domain",
		ResourceID: domain.ID, Result: "success",
	})
	writeDomainJob(w, domain, job)
}

func (h *handler) setDomain(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.UpdateDomainRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	if (request.Enabled == nil) == (request.Name == nil) {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "Change either name or enabled")
		return
	}
	var (
		domain domains.Domain
		job    jobs.Job
		err    error
		action = "domain.update"
	)
	if request.Name != nil {
		domain, job, err = h.options.Domains.RenameDomain(
			r.Context(), r.PathValue("domainId"), *request.Name,
			session.User.ID, session.User.Role == "admin",
		)
	} else {
		domain, job, err = h.options.Domains.SetDomain(
			r.Context(), r.PathValue("domainId"), session.User.ID,
			session.User.Role == "admin", *request.Enabled,
		)
		action = "domain.disable"
		if *request.Enabled {
			action = "domain.enable"
		}
	}
	if err != nil {
		h.record(r, audit.Event{
			UserID: session.User.ID, Action: action, ResourceType: "domain",
			ResourceID: r.PathValue("domainId"), Result: "failure",
		})
		h.writeDomainError(w, r, err)
		return
	}
	h.record(r, audit.Event{
		UserID: session.User.ID, Action: action, ResourceType: "domain",
		ResourceID: domain.ID, Result: "success",
	})
	writeDomainJob(w, domain, job)
}

func (h *handler) deleteDomain(w http.ResponseWriter, r *http.Request, session auth.Session) {
	domain, job, err := h.options.Domains.DeleteDomain(
		r.Context(), r.PathValue("domainId"), session.User.ID,
		session.User.Role == "admin",
	)
	if err != nil {
		h.record(r, audit.Event{
			UserID: session.User.ID, Action: "domain.delete", ResourceType: "domain",
			ResourceID: r.PathValue("domainId"), Result: "failure",
		})
		h.writeDomainError(w, r, err)
		return
	}
	h.record(r, audit.Event{
		UserID: session.User.ID, Action: "domain.delete", ResourceType: "domain",
		ResourceID: domain.ID, Result: "success",
	})
	writeDomainJob(w, domain, job)
}

func (h *handler) listAliases(w http.ResponseWriter, r *http.Request, session auth.Session) {
	query, ok := requestPage(w, r)
	if !ok {
		return
	}
	page, err := h.options.Domains.AliasPage(
		r.Context(), r.PathValue("domainId"), session.User.ID,
		session.User.Role == "admin", query,
	)
	if err != nil {
		h.writeDomainError(w, r, err)
		return
	}
	response := publicapi.DomainAliasListResponse{
		Items:      make([]publicapi.DomainAlias, 0, len(page.Items)),
		Pagination: paginationResponse(page.Query, page.Total),
	}
	for _, item := range page.Items {
		response.Items = append(response.Items, aliasResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) createAlias(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.CreateDomainAliasRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	alias, job, err := h.options.Domains.CreateAlias(
		r.Context(), r.PathValue("domainId"), request.Name,
		session.User.ID, session.User.Role == "admin",
	)
	if err != nil {
		h.record(r, audit.Event{
			UserID: session.User.ID, Action: "alias.create", ResourceType: "domain_alias",
			Result: "failure",
		})
		h.writeDomainError(w, r, err)
		return
	}
	h.record(r, audit.Event{
		UserID: session.User.ID, Action: "alias.create", ResourceType: "domain_alias",
		ResourceID: alias.ID, Result: "success",
	})
	writeAliasJob(w, alias, job)
}

func (h *handler) setAlias(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.UpdateDomainRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	if (request.Enabled == nil) == (request.Name == nil) {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "Change either name or enabled")
		return
	}
	var (
		alias  domains.Alias
		job    jobs.Job
		err    error
		action = "alias.update"
	)
	if request.Name != nil {
		alias, job, err = h.options.Domains.RenameAlias(
			r.Context(), r.PathValue("domainId"), r.PathValue("aliasId"),
			*request.Name, session.User.ID, session.User.Role == "admin",
		)
	} else {
		alias, job, err = h.options.Domains.SetAlias(
			r.Context(), r.PathValue("domainId"), r.PathValue("aliasId"),
			session.User.ID, session.User.Role == "admin", *request.Enabled,
		)
		action = "alias.disable"
		if *request.Enabled {
			action = "alias.enable"
		}
	}
	if err != nil {
		h.record(r, audit.Event{
			UserID: session.User.ID, Action: action, ResourceType: "domain_alias",
			ResourceID: r.PathValue("aliasId"), Result: "failure",
		})
		h.writeDomainError(w, r, err)
		return
	}
	h.record(r, audit.Event{
		UserID: session.User.ID, Action: action, ResourceType: "domain_alias",
		ResourceID: alias.ID, Result: "success",
	})
	writeAliasJob(w, alias, job)
}

func (h *handler) deleteAlias(w http.ResponseWriter, r *http.Request, session auth.Session) {
	alias, job, err := h.options.Domains.DeleteAlias(
		r.Context(), r.PathValue("domainId"), r.PathValue("aliasId"),
		session.User.ID, session.User.Role == "admin",
	)
	if err != nil {
		h.record(r, audit.Event{
			UserID: session.User.ID, Action: "alias.delete", ResourceType: "domain_alias",
			ResourceID: r.PathValue("aliasId"), Result: "failure",
		})
		h.writeDomainError(w, r, err)
		return
	}
	h.record(r, audit.Event{
		UserID: session.User.ID, Action: "alias.delete", ResourceType: "domain_alias",
		ResourceID: alias.ID, Result: "success",
	})
	writeAliasJob(w, alias, job)
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
		ResourceID: nodeID, Result: "success",
	})
	httpx.WriteJSON(w, http.StatusAccepted, jobResponse(job))
}

func (h *handler) listJobs(w http.ResponseWriter, r *http.Request, _ auth.Session) {
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

func (h *handler) getJob(w http.ResponseWriter, r *http.Request, _ auth.Session) {
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
	return publicapi.Node{
		Id: node.ID, Name: node.Name, Kind: publicapi.NodeKind(node.Kind),
		Endpoint: node.Endpoint, Status: publicapi.NodeStatus(node.Status),
		LastSeenAt: node.LastSeenAt, CreatedAt: node.CreatedAt, UpdatedAt: node.UpdatedAt,
	}
}

func accountResponse(account accounts.Account) publicapi.Account {
	return publicapi.Account{
		Id: account.ID, NodeId: account.NodeID, Name: account.Name,
		SystemUser: account.SystemUser, Status: publicapi.AccountStatus(account.Status),
		Enabled:   account.Enabled,
		CreatedAt: account.CreatedAt, UpdatedAt: account.UpdatedAt,
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

func domainResponse(domain domains.Domain) publicapi.Domain {
	return publicapi.Domain{
		Id: domain.ID, AccountId: domain.AccountID, NodeId: domain.NodeID,
		Name: domain.Name, Status: publicapi.DomainStatus(domain.Status),
		PhpVersion: domain.PHPVersion, Enabled: domain.Enabled,
		CreatedAt: domain.CreatedAt, UpdatedAt: domain.UpdatedAt,
	}
}

func aliasResponse(alias domains.Alias) publicapi.DomainAlias {
	return publicapi.DomainAlias{
		Id: alias.ID, DomainId: alias.DomainID, Name: alias.Name,
		Status: publicapi.DomainAliasStatus(alias.Status), Enabled: alias.Enabled,
		CreatedAt: alias.CreatedAt, UpdatedAt: alias.UpdatedAt,
	}
}

func (h *handler) writeDomainError(w http.ResponseWriter, r *http.Request, err error) {
	if writeValidationError(w, err) {
		return
	}
	switch {
	case errors.Is(err, accounts.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "Account access is required")
	case errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, "not_found", "Domain or account not found")
	case errors.Is(err, domains.ErrAccountInactive):
		writeError(w, http.StatusConflict, "account_inactive", "The account must be active")
	case errors.Is(err, domains.ErrDomainInactive):
		writeError(w, http.StatusConflict, "domain_inactive", "The domain must be active")
	case errors.Is(err, domains.ErrBusy):
		writeError(w, http.StatusConflict, "resource_busy", "A resource operation is already pending")
	case errors.Is(err, domains.ErrAliasNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Domain alias not found")
	case errors.Is(err, domains.ErrNameExists):
		writeError(w, http.StatusConflict, "domain_name_exists", "This domain name is already in use")
	case errors.Is(err, domains.ErrNameUnchanged):
		writeError(w, http.StatusConflict, "domain_name_unchanged", "The new domain name is unchanged")
	default:
		h.internalError(w, r, err)
	}
}

func writeDomainJob(w http.ResponseWriter, domain domains.Domain, job jobs.Job) {
	httpx.WriteJSON(w, http.StatusAccepted, publicapi.DomainJobResponse{
		Domain: domainResponse(domain), Job: jobResponse(job),
	})
}

func writeAliasJob(w http.ResponseWriter, alias domains.Alias, job jobs.Job) {
	httpx.WriteJSON(w, http.StatusAccepted, publicapi.DomainAliasJobResponse{
		Alias: aliasResponse(alias), Job: jobResponse(job),
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
