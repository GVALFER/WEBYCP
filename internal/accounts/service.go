package accounts

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/GVALFER/WEBYCP/internal/idgen"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/nodes"
	"github.com/GVALFER/WEBYCP/internal/packages"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

type Service struct {
	repository Repository
	nodes      nodes.Repository
	agent      Agent
	packages   *packages.Service
	notify     func()
}

type createPayload struct {
	AccountID string `json:"accountId"`
}

func NewService(repository Repository, nodeRepository nodes.Repository, agent Agent, packageService *packages.Service, notify func()) *Service {
	return &Service{
		repository: repository,
		nodes:      nodeRepository,
		agent:      agent,
		packages:   packageService,
		notify:     notify,
	}
}

func (s *Service) Create(
	ctx context.Context,
	name, nodeID, packageID, ownerID string,
) (Account, jobs.Job, error) {
	normalizedName, err := validate.AccountName(name)
	if err != nil {
		return Account{}, jobs.Job{}, err
	}
	if nodeID == "" {
		return Account{}, jobs.Job{}, &validate.Error{
			Field: "nodeId", Message: "Select a managed node",
		}
	}
	if _, err := s.nodes.Node(ctx, nodeID); err != nil {
		return Account{}, jobs.Job{}, fmt.Errorf("get account node: %w", err)
	}
	if _, err := s.packages.Package(ctx, packageID); err != nil {
		return Account{}, jobs.Job{}, fmt.Errorf("get account package: %w", err)
	}

	accountID, err := idgen.ID()
	if err != nil {
		return Account{}, jobs.Job{}, err
	}
	jobID, err := idgen.ID()
	if err != nil {
		return Account{}, jobs.Job{}, err
	}
	payload, err := json.Marshal(createPayload{AccountID: accountID})
	if err != nil {
		return Account{}, jobs.Job{}, err
	}
	now := time.Now().UTC()
	account := Account{
		ID: accountID, NodeID: nodeID, Name: normalizedName,
		SystemUser: "wcp_" + accountID[:12], Status: "pending", Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}
	job := jobs.Job{
		ID: jobID, NodeID: nodeID, UserID: ownerID, Kind: jobs.KindAccountCreate,
		Status: "queued", Payload: string(payload), MaxAttempts: 2, CreatedAt: now,
	}
	account, job, err = s.repository.CreateProvision(ctx, account, ownerID, packageID, job)
	if err != nil {
		return Account{}, jobs.Job{}, fmt.Errorf("create account provision: %w", err)
	}

	s.notify()
	return account, job, nil
}

func (s *Service) Set(
	ctx context.Context, id, userID string, admin, enabled bool,
) (Account, jobs.Job, error) {
	account, err := s.Account(ctx, id, userID, admin)
	if err != nil {
		return Account{}, jobs.Job{}, err
	}
	if account.Status == "pending" {
		return Account{}, jobs.Job{}, ErrBusy
	}
	kind := jobs.KindAccountDisable
	if enabled {
		kind = jobs.KindAccountEnable
	}
	job, err := newActionJob(account, userID, kind)
	if err != nil {
		return Account{}, jobs.Job{}, err
	}
	account, job, err = s.repository.QueueAction(ctx, account.ID, enabled, job)
	if err == nil {
		s.notify()
	}
	return account, job, err
}

func (s *Service) Delete(
	ctx context.Context, id, userID string, admin bool,
) (Account, jobs.Job, error) {
	account, err := s.Account(ctx, id, userID, admin)
	if err != nil {
		return Account{}, jobs.Job{}, err
	}
	if account.Status == "pending" {
		return Account{}, jobs.Job{}, ErrBusy
	}
	count, err := s.repository.ResourceCount(ctx, account.ID)
	if err != nil {
		return Account{}, jobs.Job{}, fmt.Errorf("count account resources: %w", err)
	}
	if count != 0 {
		return Account{}, jobs.Job{}, ErrNotEmpty
	}
	job, err := newActionJob(account, userID, jobs.KindAccountDelete)
	if err != nil {
		return Account{}, jobs.Job{}, err
	}
	account, job, err = s.repository.QueueAction(ctx, account.ID, false, job)
	if err == nil {
		s.notify()
	}
	return account, job, err
}

func newActionJob(account Account, userID, kind string) (jobs.Job, error) {
	id, err := idgen.ID()
	if err != nil {
		return jobs.Job{}, err
	}
	payload, err := json.Marshal(createPayload{AccountID: account.ID})
	if err != nil {
		return jobs.Job{}, err
	}
	return jobs.Job{
		ID: id, NodeID: account.NodeID, UserID: userID, Kind: kind,
		Status: "queued", Payload: string(payload), MaxAttempts: 2,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (s *Service) Accounts(ctx context.Context, userID string, admin bool) ([]Account, error) {
	return s.repository.Accounts(ctx, userID, admin)
}

func (s *Service) AccountPage(
	ctx context.Context, userID string, admin bool, query pagination.Query,
) (pagination.Result[Overview], error) {
	return s.repository.AccountPage(ctx, userID, admin, query)
}

func (s *Service) AssignPackage(ctx context.Context, id, packageID, userID string, admin bool) (Overview, error) {
	if _, err := s.Account(ctx, id, userID, admin); err != nil {
		return Overview{}, err
	}
	if err := s.packages.Assign(ctx, id, packageID); err != nil {
		return Overview{}, err
	}
	return s.repository.AccountOverview(ctx, id)
}

func (s *Service) Account(ctx context.Context, id, userID string, admin bool) (Account, error) {
	account, err := s.repository.Account(ctx, id)
	if err != nil {
		return Account{}, err
	}
	if admin {
		return account, nil
	}
	member, err := s.repository.AccountMember(ctx, id, userID)
	if err != nil {
		return Account{}, err
	}
	if !member {
		return Account{}, ErrForbidden
	}
	return account, nil
}

func (s *Service) Get(ctx context.Context, id string) (Account, error) {
	return s.repository.Account(ctx, id)
}

func (s *Service) Provision(ctx context.Context, job jobs.Job) error {
	var payload createPayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil || payload.AccountID == "" {
		return fmt.Errorf("decode account job payload")
	}
	account, err := s.repository.Account(ctx, payload.AccountID)
	if err != nil {
		return fmt.Errorf("get account: %w", err)
	}
	node, err := s.nodes.Node(ctx, account.NodeID)
	if err != nil {
		return fmt.Errorf("get account node: %w", err)
	}
	if err := s.repository.UpdateStatus(ctx, account.ID, "pending"); err != nil {
		return fmt.Errorf("mark account pending: %w", err)
	}
	if err := s.agent.EnsureAccount(ctx, node.Endpoint, account.ID, account.SystemUser); err != nil {
		if updateErr := s.repository.UpdateStatus(ctx, account.ID, "error"); updateErr != nil {
			return fmt.Errorf("ensure account: %v; mark account failed: %w", err, updateErr)
		}
		return fmt.Errorf("ensure account: %w", err)
	}
	if err := s.repository.UpdateStatus(ctx, account.ID, "active"); err != nil {
		return fmt.Errorf("mark account active: %w", err)
	}

	return nil
}

func (s *Service) ProvisionAction(ctx context.Context, job jobs.Job) error {
	var payload createPayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil || payload.AccountID == "" {
		return fmt.Errorf("decode account action payload")
	}
	account, err := s.repository.Account(ctx, payload.AccountID)
	if err != nil {
		return fmt.Errorf("get account: %w", err)
	}
	node, err := s.nodes.Node(ctx, account.NodeID)
	if err != nil {
		return fmt.Errorf("get account node: %w", err)
	}
	var operation func(context.Context, string, string, string) error
	status := "disabled"
	switch job.Kind {
	case jobs.KindAccountDisable:
		operation = s.agent.DisableAccount
	case jobs.KindAccountEnable:
		operation = s.agent.EnableAccount
		status = "active"
	case jobs.KindAccountDelete:
		operation = s.agent.DeleteAccount
	default:
		return fmt.Errorf("unsupported account action %q", job.Kind)
	}
	if err := operation(ctx, node.Endpoint, account.ID, account.SystemUser); err != nil {
		if updateErr := s.repository.UpdateStatus(ctx, account.ID, "error"); updateErr != nil {
			return fmt.Errorf("account action: %v; mark account failed: %w", err, updateErr)
		}
		return fmt.Errorf("account action: %w", err)
	}
	if job.Kind == jobs.KindAccountDelete {
		return s.repository.Delete(ctx, account.ID)
	}
	return s.repository.UpdateStatus(ctx, account.ID, status)
}
