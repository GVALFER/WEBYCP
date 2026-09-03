package domains

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/idgen"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/nodes"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

type Service struct {
	repository Repository
	accounts   *accounts.Service
	nodes      nodes.Repository
	agent      Agent
	notify     func()
}

type createPayload struct {
	DomainID string `json:"domainId"`
}

type aliasPayload struct {
	AliasID string `json:"aliasId"`
}

type domainRenamePayload struct {
	DomainID     string `json:"domainId"`
	PreviousName string `json:"previousName"`
	Name         string `json:"name"`
}

type aliasRenamePayload struct {
	AliasID      string `json:"aliasId"`
	PreviousName string `json:"previousName"`
	Name         string `json:"name"`
}

func NewService(
	repository Repository,
	accountService *accounts.Service,
	nodeRepository nodes.Repository,
	agent Agent,
	notify func(),
) *Service {
	return &Service{
		repository: repository, accounts: accountService, nodes: nodeRepository,
		agent: agent, notify: notify,
	}
}

func (s *Service) Create(
	ctx context.Context,
	accountID, name, userID string,
	admin bool,
) (Domain, jobs.Job, error) {
	if err := validate.ID("accountId", accountID); err != nil {
		return Domain{}, jobs.Job{}, err
	}
	normalizedName, err := validate.Domain(name)
	if err != nil {
		return Domain{}, jobs.Job{}, err
	}
	account, err := s.accounts.Account(ctx, accountID, userID, admin)
	if err != nil {
		return Domain{}, jobs.Job{}, err
	}
	if account.Status != "active" {
		return Domain{}, jobs.Job{}, ErrAccountInactive
	}

	domainID, err := idgen.ID()
	if err != nil {
		return Domain{}, jobs.Job{}, err
	}
	jobID, err := idgen.ID()
	if err != nil {
		return Domain{}, jobs.Job{}, err
	}
	payload, err := json.Marshal(createPayload{DomainID: domainID})
	if err != nil {
		return Domain{}, jobs.Job{}, err
	}
	now := time.Now().UTC()
	domain := Domain{
		ID: domainID, AccountID: account.ID, NodeID: account.NodeID,
		Name: normalizedName, Status: "pending", PHPVersion: "8.3",
		CreatedAt: now, UpdatedAt: now,
	}
	job := jobs.Job{
		ID: jobID, NodeID: account.NodeID, UserID: userID, Kind: jobs.KindDomainCreate,
		Status: "queued", Payload: string(payload), MaxAttempts: 2, CreatedAt: now,
	}
	domain, job, err = s.repository.CreateDomainProvision(ctx, domain, job)
	if err != nil {
		return Domain{}, jobs.Job{}, fmt.Errorf("create domain provision: %w", err)
	}

	s.notify()
	return domain, job, nil
}

func (s *Service) Domains(ctx context.Context, userID string, admin bool) ([]Domain, error) {
	return s.repository.Domains(ctx, userID, admin)
}

func (s *Service) GetDomain(ctx context.Context, id, userID string, admin bool) (Domain, error) {
	return s.domain(ctx, id, userID, admin)
}

func (s *Service) Get(ctx context.Context, id string) (Domain, error) {
	return s.repository.Domain(ctx, id)
}

func (s *Service) CreateAlias(
	ctx context.Context,
	domainID, name, userID string,
	admin bool,
) (Alias, jobs.Job, error) {
	domain, err := s.domain(ctx, domainID, userID, admin)
	if err != nil {
		return Alias{}, jobs.Job{}, err
	}
	if domain.Status != "active" {
		return Alias{}, jobs.Job{}, ErrDomainInactive
	}
	existing, err := s.repository.Aliases(ctx, domain.ID)
	if err != nil {
		return Alias{}, jobs.Job{}, fmt.Errorf("list domain aliases: %w", err)
	}
	if len(existing) >= validate.MaxDomainAliases {
		return Alias{}, jobs.Job{}, &validate.Error{
			Field: "name", Message: "This domain has reached its alias limit",
		}
	}
	normalizedName, err := validate.Domain(name)
	if err != nil {
		return Alias{}, jobs.Job{}, err
	}
	aliasID, err := idgen.ID()
	if err != nil {
		return Alias{}, jobs.Job{}, err
	}
	jobID, err := idgen.ID()
	if err != nil {
		return Alias{}, jobs.Job{}, err
	}
	payload, err := json.Marshal(aliasPayload{AliasID: aliasID})
	if err != nil {
		return Alias{}, jobs.Job{}, err
	}
	now := time.Now().UTC()
	alias := Alias{
		ID: aliasID, DomainID: domain.ID, Name: normalizedName, Status: "pending",
		CreatedAt: now, UpdatedAt: now,
	}
	job := jobs.Job{
		ID: jobID, NodeID: domain.NodeID, UserID: userID, Kind: jobs.KindAliasCreate,
		Status: "queued", Payload: string(payload), MaxAttempts: 2, CreatedAt: now,
	}
	alias, job, err = s.repository.CreateAliasProvision(ctx, alias, job)
	if err != nil {
		return Alias{}, jobs.Job{}, fmt.Errorf("create alias provision: %w", err)
	}
	s.notify()
	return alias, job, nil
}

func (s *Service) Aliases(
	ctx context.Context,
	domainID, userID string,
	admin bool,
) ([]Alias, error) {
	domain, err := s.domain(ctx, domainID, userID, admin)
	if err != nil {
		return nil, err
	}
	return s.repository.Aliases(ctx, domain.ID)
}

func (s *Service) SetDomain(
	ctx context.Context,
	domainID, userID string,
	admin, enabled bool,
) (Domain, jobs.Job, error) {
	domain, err := s.domain(ctx, domainID, userID, admin)
	if err != nil {
		return Domain{}, jobs.Job{}, err
	}
	if domain.Status == "pending" {
		return Domain{}, jobs.Job{}, ErrBusy
	}
	kind := jobs.KindDomainDisable
	if enabled {
		kind = jobs.KindDomainEnable
	}
	job, err := actionJob(domain.NodeID, userID, kind, createPayload{DomainID: domain.ID})
	if err != nil {
		return Domain{}, jobs.Job{}, err
	}
	domain, job, err = s.repository.QueueDomainAction(ctx, domain.ID, enabled, job)
	if err == nil {
		s.notify()
	}
	return domain, job, err
}

func (s *Service) DeleteDomain(
	ctx context.Context,
	domainID, userID string,
	admin bool,
) (Domain, jobs.Job, error) {
	domain, err := s.domain(ctx, domainID, userID, admin)
	if err != nil {
		return Domain{}, jobs.Job{}, err
	}
	if domain.Status == "pending" {
		return Domain{}, jobs.Job{}, ErrBusy
	}
	job, err := actionJob(
		domain.NodeID, userID, jobs.KindDomainDelete, createPayload{DomainID: domain.ID},
	)
	if err != nil {
		return Domain{}, jobs.Job{}, err
	}
	domain, job, err = s.repository.QueueDomainAction(ctx, domain.ID, false, job)
	if err == nil {
		s.notify()
	}
	return domain, job, err
}

func (s *Service) SetAlias(
	ctx context.Context,
	domainID, aliasID, userID string,
	admin, enabled bool,
) (Alias, jobs.Job, error) {
	domain, alias, err := s.alias(ctx, domainID, aliasID, userID, admin)
	if err != nil {
		return Alias{}, jobs.Job{}, err
	}
	if domain.Status != "active" {
		return Alias{}, jobs.Job{}, ErrDomainInactive
	}
	if alias.Status == "pending" {
		return Alias{}, jobs.Job{}, ErrBusy
	}
	kind := jobs.KindAliasDisable
	if enabled {
		kind = jobs.KindAliasEnable
	}
	job, err := actionJob(domain.NodeID, userID, kind, aliasPayload{AliasID: alias.ID})
	if err != nil {
		return Alias{}, jobs.Job{}, err
	}
	alias, job, err = s.repository.QueueAliasAction(ctx, alias.ID, enabled, job)
	if err == nil {
		s.notify()
	}
	return alias, job, err
}

func (s *Service) DeleteAlias(
	ctx context.Context,
	domainID, aliasID, userID string,
	admin bool,
) (Alias, jobs.Job, error) {
	domain, alias, err := s.alias(ctx, domainID, aliasID, userID, admin)
	if err != nil {
		return Alias{}, jobs.Job{}, err
	}
	if domain.Status != "active" {
		return Alias{}, jobs.Job{}, ErrDomainInactive
	}
	if alias.Status == "pending" {
		return Alias{}, jobs.Job{}, ErrBusy
	}
	job, err := actionJob(
		domain.NodeID, userID, jobs.KindAliasDelete, aliasPayload{AliasID: alias.ID},
	)
	if err != nil {
		return Alias{}, jobs.Job{}, err
	}
	alias, job, err = s.repository.QueueAliasAction(ctx, alias.ID, false, job)
	if err == nil {
		s.notify()
	}
	return alias, job, err
}

func (s *Service) RenameDomain(
	ctx context.Context,
	domainID, name, userID string,
	admin bool,
) (Domain, jobs.Job, error) {
	domain, err := s.domain(ctx, domainID, userID, admin)
	if err != nil {
		return Domain{}, jobs.Job{}, err
	}
	if domain.Status != "active" {
		return Domain{}, jobs.Job{}, ErrDomainInactive
	}
	name, err = renameName(domain.Name, name)
	if err != nil {
		return Domain{}, jobs.Job{}, err
	}
	job, err := actionJob(domain.NodeID, userID, jobs.KindDomainUpdate, domainRenamePayload{
		DomainID: domain.ID, PreviousName: domain.Name, Name: name,
	})
	if err != nil {
		return Domain{}, jobs.Job{}, err
	}
	domain, job, err = s.repository.QueueDomainRename(ctx, domain.ID, name, job)
	if err == nil {
		s.notify()
	}
	return domain, job, err
}

func (s *Service) RenameAlias(
	ctx context.Context,
	domainID, aliasID, name, userID string,
	admin bool,
) (Alias, jobs.Job, error) {
	domain, alias, err := s.alias(ctx, domainID, aliasID, userID, admin)
	if err != nil {
		return Alias{}, jobs.Job{}, err
	}
	if domain.Status != "active" {
		return Alias{}, jobs.Job{}, ErrDomainInactive
	}
	if alias.Status == "pending" {
		return Alias{}, jobs.Job{}, ErrBusy
	}
	name, err = renameName(alias.Name, name)
	if err != nil {
		return Alias{}, jobs.Job{}, err
	}
	job, err := actionJob(domain.NodeID, userID, jobs.KindAliasUpdate, aliasRenamePayload{
		AliasID: alias.ID, PreviousName: alias.Name, Name: name,
	})
	if err != nil {
		return Alias{}, jobs.Job{}, err
	}
	alias, job, err = s.repository.QueueAliasRename(ctx, alias.ID, name, job)
	if err == nil {
		s.notify()
	}
	return alias, job, err
}

func (s *Service) Provision(ctx context.Context, job jobs.Job) error {
	var payload createPayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil || payload.DomainID == "" {
		return fmt.Errorf("decode domain job payload")
	}
	domain, err := s.repository.Domain(ctx, payload.DomainID)
	if err != nil {
		return fmt.Errorf("get domain: %w", err)
	}
	if err := s.repository.UpdateDomainStatus(ctx, domain.ID, "pending"); err != nil {
		return fmt.Errorf("mark domain pending: %w", err)
	}
	if err := s.ensure(ctx, domain); err != nil {
		if updateErr := s.repository.UpdateDomainStatus(ctx, domain.ID, "error"); updateErr != nil {
			return fmt.Errorf("ensure domain: %v; mark domain failed: %w", err, updateErr)
		}
		return fmt.Errorf("ensure domain: %w", err)
	}
	if err := s.repository.UpdateDomainStatus(ctx, domain.ID, "active"); err != nil {
		return fmt.Errorf("mark domain active: %w", err)
	}
	return nil
}

func (s *Service) ProvisionAlias(ctx context.Context, job jobs.Job) error {
	var payload aliasPayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil || payload.AliasID == "" {
		return fmt.Errorf("decode alias job payload")
	}
	alias, err := s.repository.Alias(ctx, payload.AliasID)
	if err != nil {
		return fmt.Errorf("get domain alias: %w", err)
	}
	domain, err := s.repository.Domain(ctx, alias.DomainID)
	if err != nil {
		return fmt.Errorf("get alias domain: %w", err)
	}
	if err := s.repository.UpdateAliasStatus(ctx, alias.ID, "pending"); err != nil {
		return fmt.Errorf("mark alias pending: %w", err)
	}
	if err := s.ensure(ctx, domain); err != nil {
		if updateErr := s.repository.UpdateAliasStatus(ctx, alias.ID, "error"); updateErr != nil {
			return fmt.Errorf("ensure domain alias: %v; mark alias failed: %w", err, updateErr)
		}
		return fmt.Errorf("ensure domain alias: %w", err)
	}
	if err := s.repository.UpdateAliasStatus(ctx, alias.ID, "active"); err != nil {
		return fmt.Errorf("mark alias active: %w", err)
	}
	return nil
}

func (s *Service) ProvisionDomainAction(ctx context.Context, job jobs.Job) error {
	if job.Kind != jobs.KindDomainEnable &&
		job.Kind != jobs.KindDomainDisable &&
		job.Kind != jobs.KindDomainDelete {
		return fmt.Errorf("unsupported domain action: %s", job.Kind)
	}
	var payload createPayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil || payload.DomainID == "" {
		return fmt.Errorf("decode domain action payload")
	}
	domain, err := s.repository.Domain(ctx, payload.DomainID)
	if err != nil {
		if job.Kind == jobs.KindDomainDelete && errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("get domain: %w", err)
	}
	account, node, err := s.host(ctx, domain)
	if err != nil {
		return err
	}
	switch job.Kind {
	case jobs.KindDomainEnable:
		err = s.ensure(ctx, domain)
	case jobs.KindDomainDisable:
		err = s.agent.DisableDomain(
			ctx, node.Endpoint, account.ID, account.SystemUser, domain.ID,
		)
	case jobs.KindDomainDelete:
		err = s.agent.DeleteDomain(
			ctx, node.Endpoint, account.ID, account.SystemUser, domain.ID, domain.Name,
		)
	}
	if err != nil {
		if updateErr := s.repository.UpdateDomainStatus(ctx, domain.ID, "error"); updateErr != nil {
			return fmt.Errorf("reconcile domain action: %v; mark domain failed: %w", err, updateErr)
		}
		return fmt.Errorf("reconcile domain action: %w", err)
	}
	if job.Kind == jobs.KindDomainDelete {
		if err := s.repository.DeleteDomain(ctx, domain.ID); err != nil {
			return fmt.Errorf("delete domain metadata: %w", err)
		}
		return nil
	}
	status := "disabled"
	if job.Kind == jobs.KindDomainEnable {
		status = "active"
	}
	if err := s.repository.UpdateDomainStatus(ctx, domain.ID, status); err != nil {
		return fmt.Errorf("mark domain %s: %w", status, err)
	}
	return nil
}

func (s *Service) ProvisionAliasAction(ctx context.Context, job jobs.Job) error {
	if job.Kind != jobs.KindAliasEnable &&
		job.Kind != jobs.KindAliasDisable &&
		job.Kind != jobs.KindAliasDelete {
		return fmt.Errorf("unsupported alias action: %s", job.Kind)
	}
	var payload aliasPayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil || payload.AliasID == "" {
		return fmt.Errorf("decode alias action payload")
	}
	alias, err := s.repository.Alias(ctx, payload.AliasID)
	if err != nil {
		if job.Kind == jobs.KindAliasDelete && errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("get domain alias: %w", err)
	}
	domain, err := s.repository.Domain(ctx, alias.DomainID)
	if err != nil {
		return fmt.Errorf("get alias domain: %w", err)
	}
	if err := s.ensure(ctx, domain); err != nil {
		if updateErr := s.repository.UpdateAliasStatus(ctx, alias.ID, "error"); updateErr != nil {
			return fmt.Errorf("reconcile alias action: %v; mark alias failed: %w", err, updateErr)
		}
		return fmt.Errorf("reconcile alias action: %w", err)
	}
	if job.Kind == jobs.KindAliasDelete {
		if err := s.repository.DeleteAlias(ctx, alias.ID); err != nil {
			return fmt.Errorf("delete alias metadata: %w", err)
		}
		return nil
	}
	status := "disabled"
	if job.Kind == jobs.KindAliasEnable {
		status = "active"
	}
	if err := s.repository.UpdateAliasStatus(ctx, alias.ID, status); err != nil {
		return fmt.Errorf("mark alias %s: %w", status, err)
	}
	return nil
}

func (s *Service) ProvisionDomainRename(ctx context.Context, job jobs.Job) error {
	if job.Kind != jobs.KindDomainUpdate {
		return fmt.Errorf("unsupported domain rename: %s", job.Kind)
	}
	var payload domainRenamePayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil ||
		payload.DomainID == "" || payload.PreviousName == "" || payload.Name == "" {
		return fmt.Errorf("decode domain rename payload")
	}
	domain, err := s.repository.Domain(ctx, payload.DomainID)
	if err != nil {
		return fmt.Errorf("get domain: %w", err)
	}
	if domain.PreviousName == "" && domain.Name == payload.Name && domain.Status == "active" {
		return nil
	}
	if domain.Name != payload.Name || domain.PreviousName != payload.PreviousName {
		return fmt.Errorf("domain rename state does not match job")
	}
	account, node, err := s.host(ctx, domain)
	if err != nil {
		return s.domainRenameError(ctx, job, domain.ID, err)
	}
	aliases, err := s.repository.EnabledAliases(ctx, domain.ID)
	if err != nil {
		return s.domainRenameError(ctx, job, domain.ID, fmt.Errorf("list domain aliases: %w", err))
	}
	names := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		names = append(names, alias.Name)
	}
	err = s.agent.RenameDomain(
		ctx, node.Endpoint, account.ID, account.SystemUser, domain.ID,
		payload.PreviousName, payload.Name, domain.PHPVersion, names,
	)
	if err != nil {
		return s.domainRenameError(ctx, job, domain.ID, err)
	}
	if err := s.repository.CompleteDomainRename(ctx, domain.ID); err != nil {
		return fmt.Errorf("complete domain rename: %w", err)
	}
	return nil
}

func (s *Service) ProvisionAliasRename(ctx context.Context, job jobs.Job) error {
	if job.Kind != jobs.KindAliasUpdate {
		return fmt.Errorf("unsupported alias rename: %s", job.Kind)
	}
	var payload aliasRenamePayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil ||
		payload.AliasID == "" || payload.PreviousName == "" || payload.Name == "" {
		return fmt.Errorf("decode alias rename payload")
	}
	alias, err := s.repository.Alias(ctx, payload.AliasID)
	if err != nil {
		return fmt.Errorf("get domain alias: %w", err)
	}
	if alias.PreviousName == "" && alias.Name == payload.Name &&
		(alias.Status == "active" || alias.Status == "disabled") {
		return nil
	}
	if alias.Name != payload.Name || alias.PreviousName != payload.PreviousName {
		return fmt.Errorf("alias rename state does not match job")
	}
	if alias.Enabled {
		domain, err := s.repository.Domain(ctx, alias.DomainID)
		if err != nil {
			return s.aliasRenameError(ctx, job, alias.ID, fmt.Errorf("get alias domain: %w", err))
		}
		if err := s.ensure(ctx, domain); err != nil {
			return s.aliasRenameError(ctx, job, alias.ID, err)
		}
	}
	if err := s.repository.CompleteAliasRename(ctx, alias.ID); err != nil {
		return fmt.Errorf("complete alias rename: %w", err)
	}
	return nil
}

func (s *Service) domain(
	ctx context.Context,
	domainID, userID string,
	admin bool,
) (Domain, error) {
	if err := validate.ID("domainId", domainID); err != nil {
		return Domain{}, err
	}
	domain, err := s.repository.Domain(ctx, domainID)
	if err != nil {
		return Domain{}, err
	}
	if _, err := s.accounts.Account(ctx, domain.AccountID, userID, admin); err != nil {
		return Domain{}, err
	}
	return domain, nil
}

func (s *Service) alias(
	ctx context.Context,
	domainID, aliasID, userID string,
	admin bool,
) (Domain, Alias, error) {
	domain, err := s.domain(ctx, domainID, userID, admin)
	if err != nil {
		return Domain{}, Alias{}, err
	}
	if err := validate.ID("aliasId", aliasID); err != nil {
		return Domain{}, Alias{}, err
	}
	alias, err := s.repository.Alias(ctx, aliasID)
	if err != nil {
		return Domain{}, Alias{}, err
	}
	if alias.DomainID != domain.ID {
		return Domain{}, Alias{}, ErrAliasNotFound
	}
	return domain, alias, nil
}

func (s *Service) ensure(ctx context.Context, domain Domain) error {
	account, node, err := s.host(ctx, domain)
	if err != nil {
		return err
	}
	aliases, err := s.repository.EnabledAliases(ctx, domain.ID)
	if err != nil {
		return fmt.Errorf("list domain aliases: %w", err)
	}
	names := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		names = append(names, alias.Name)
	}
	return s.agent.EnsureDomain(
		ctx, node.Endpoint, account.ID, account.SystemUser,
		domain.ID, domain.Name, domain.PHPVersion, names,
	)
}

func (s *Service) host(ctx context.Context, domain Domain) (accounts.Account, nodes.Node, error) {
	account, err := s.accounts.Get(ctx, domain.AccountID)
	if err != nil {
		return accounts.Account{}, nodes.Node{}, fmt.Errorf("get domain account: %w", err)
	}
	if account.Status != "active" {
		return accounts.Account{}, nodes.Node{}, ErrAccountInactive
	}
	node, err := s.nodes.Node(ctx, domain.NodeID)
	if err != nil {
		return accounts.Account{}, nodes.Node{}, fmt.Errorf("get domain node: %w", err)
	}
	return account, node, nil
}

func actionJob(nodeID, userID, kind string, payload any) (jobs.Job, error) {
	id, err := idgen.ID()
	if err != nil {
		return jobs.Job{}, err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return jobs.Job{}, err
	}
	return jobs.Job{
		ID: id, NodeID: nodeID, UserID: userID, Kind: kind,
		Status: "queued", Payload: string(data), MaxAttempts: 2, CreatedAt: time.Now().UTC(),
	}, nil
}

func renameName(current, next string) (string, error) {
	name, err := validate.Domain(next)
	if err != nil {
		return "", err
	}
	if name == current {
		return "", ErrNameUnchanged
	}
	return name, nil
}

func (s *Service) domainRenameError(
	ctx context.Context,
	job jobs.Job,
	domainID string,
	renameErr error,
) error {
	if job.Attempts >= job.MaxAttempts {
		if err := s.repository.FailDomainRename(ctx, domainID); err != nil {
			return fmt.Errorf("rename domain: %v; restore domain name: %w", renameErr, err)
		}
	}
	return fmt.Errorf("rename domain: %w", renameErr)
}

func (s *Service) aliasRenameError(
	ctx context.Context,
	job jobs.Job,
	aliasID string,
	renameErr error,
) error {
	if job.Attempts >= job.MaxAttempts {
		if err := s.repository.FailAliasRename(ctx, aliasID); err != nil {
			return fmt.Errorf("rename alias: %v; restore alias name: %w", renameErr, err)
		}
	}
	return fmt.Errorf("rename alias: %w", renameErr)
}
