package cronjob

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/idgen"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/nodes"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

type Service struct {
	repository Repository
	accounts   *accounts.Service
	nodes      nodes.Repository
	agent      Agent
	notify     func()
}

type syncPayload struct {
	AccountID string `json:"accountId"`
}

func NewService(repository Repository, accounts *accounts.Service, nodes nodes.Repository, agent Agent, notify func()) *Service {
	return &Service{repository: repository, accounts: accounts, nodes: nodes, agent: agent, notify: notify}
}

func (s *Service) Create(
	ctx context.Context, accountID, name, schedule, command, userID string, admin, enabled bool,
) (CronJob, jobs.Job, error) {
	account, err := s.account(ctx, accountID, userID, admin)
	if err != nil {
		return CronJob{}, jobs.Job{}, err
	}
	value, err := build(CronJob{AccountID: account.ID, NodeID: account.NodeID}, name, schedule, command, enabled)
	if err != nil {
		return CronJob{}, jobs.Job{}, err
	}
	value.ID, err = idgen.ID()
	if err != nil {
		return CronJob{}, jobs.Job{}, err
	}
	value.CreatedAt = time.Now().UTC()
	value.UpdatedAt = value.CreatedAt
	job, err := s.job(account.NodeID, userID, account.ID)
	if err != nil {
		return CronJob{}, jobs.Job{}, err
	}
	value, job, err = s.repository.CreateCronJob(ctx, value, job)
	if err == nil {
		s.notify()
	}
	return value, job, err
}

func (s *Service) Update(
	ctx context.Context, id, accountID, name, schedule, command, userID string, admin, enabled bool,
) (CronJob, jobs.Job, error) {
	current, err := s.get(ctx, id, userID, admin)
	if err != nil {
		return CronJob{}, jobs.Job{}, err
	}
	if current.AccountID != accountID {
		return CronJob{}, jobs.Job{}, accounts.ErrForbidden
	}
	value, err := build(current, name, schedule, command, enabled)
	if err != nil {
		return CronJob{}, jobs.Job{}, err
	}
	value.UpdatedAt = time.Now().UTC()
	job, err := s.job(value.NodeID, userID, value.AccountID)
	if err != nil {
		return CronJob{}, jobs.Job{}, err
	}
	value, job, err = s.repository.UpdateCronJob(ctx, value, job)
	if err == nil {
		s.notify()
	}
	return value, job, err
}

func (s *Service) Delete(ctx context.Context, id, userID string, admin bool) (jobs.Job, error) {
	value, err := s.get(ctx, id, userID, admin)
	if err != nil {
		return jobs.Job{}, err
	}
	job, err := s.job(value.NodeID, userID, value.AccountID)
	if err != nil {
		return jobs.Job{}, err
	}
	job, err = s.repository.DeleteCronJob(ctx, value.ID, job)
	if err == nil {
		s.notify()
	}
	return job, err
}

func (s *Service) CronJobs(ctx context.Context, userID string, admin bool) ([]CronJob, error) {
	return s.repository.CronJobs(ctx, userID, admin)
}

func (s *Service) CronJobPage(
	ctx context.Context, userID string, admin bool, query pagination.Query,
) (pagination.Result[CronJob], error) {
	return s.repository.CronJobPage(ctx, userID, admin, query)
}

func (s *Service) Sync(ctx context.Context, job jobs.Job) error {
	var payload syncPayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil || payload.AccountID == "" {
		return fmt.Errorf("decode cron sync payload")
	}
	return s.Reconcile(ctx, payload.AccountID)
}

func (s *Service) Reconcile(ctx context.Context, accountID string) error {
	account, err := s.accounts.Get(ctx, accountID)
	if err != nil {
		return err
	}
	node, err := s.nodes.Node(ctx, account.NodeID)
	if err != nil {
		return err
	}
	values, err := s.repository.AccountCronJobs(ctx, account.ID)
	if err != nil {
		return err
	}
	entries := make([]Entry, 0, len(values))
	for _, value := range values {
		if value.Enabled && account.Enabled && account.Status == "active" {
			entries = append(entries, Entry{ID: value.ID, Schedule: value.Schedule, Command: value.Command})
		}
	}
	if err := s.agent.SyncCron(ctx, node.Endpoint, account.ID, account.SystemUser, entries); err != nil {
		_ = s.repository.SetCronStatuses(ctx, account.ID, "error", "error")
		return err
	}
	return s.repository.SetCronStatuses(ctx, account.ID, "active", "disabled")
}

func (s *Service) get(ctx context.Context, id, userID string, admin bool) (CronJob, error) {
	if err := validate.ID("cronJobId", id); err != nil {
		return CronJob{}, err
	}
	value, err := s.repository.CronJob(ctx, id)
	if err != nil {
		return CronJob{}, err
	}
	if _, err := s.accounts.Account(ctx, value.AccountID, userID, admin); err != nil {
		return CronJob{}, err
	}
	return value, nil
}

func (s *Service) account(ctx context.Context, id, userID string, admin bool) (accounts.Account, error) {
	account, err := s.accounts.Account(ctx, id, userID, admin)
	if err != nil {
		return accounts.Account{}, err
	}
	if account.Status != "active" || !account.Enabled {
		return accounts.Account{}, accounts.ErrBusy
	}
	return account, nil
}

func (s *Service) job(nodeID, userID, accountID string) (jobs.Job, error) {
	id, err := idgen.ID()
	if err != nil {
		return jobs.Job{}, err
	}
	payload, err := json.Marshal(syncPayload{AccountID: accountID})
	if err != nil {
		return jobs.Job{}, err
	}
	return jobs.Job{ID: id, NodeID: nodeID, UserID: userID, Kind: jobs.KindCronSync, Status: "queued", Payload: string(payload), MaxAttempts: 2, CreatedAt: time.Now().UTC()}, nil
}

func build(value CronJob, name, schedule, command string, enabled bool) (CronJob, error) {
	var err error
	value.Name, err = validate.ResourceName(name)
	if err != nil {
		return CronJob{}, err
	}
	value.Schedule, err = validate.CronSchedule(schedule, false)
	if err != nil {
		return CronJob{}, err
	}
	value.Command, err = validate.CronCommand(command)
	if err != nil {
		return CronJob{}, err
	}
	value.Enabled = enabled
	value.Status = "pending"
	return value, nil
}
