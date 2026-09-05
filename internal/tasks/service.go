package tasks

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
	"github.com/GVALFER/WEBYCP/internal/services"
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
	ctx context.Context, accountID, name, schedule, command, driver, userID string, kind Kind,
	admin, enabled bool,
) (ScheduledTask, jobs.Job, error) {
	account, err := s.account(ctx, accountID, userID, admin)
	if err != nil {
		return ScheduledTask{}, jobs.Job{}, err
	}
	value, err := build(
		ScheduledTask{AccountID: account.ID, NodeID: account.NodeID, Kind: kind},
		name, schedule, command, driver, enabled,
	)
	if err != nil {
		return ScheduledTask{}, jobs.Job{}, err
	}
	value.ID, err = idgen.ID()
	if err != nil {
		return ScheduledTask{}, jobs.Job{}, err
	}
	value.CreatedAt = time.Now().UTC()
	value.UpdatedAt = value.CreatedAt
	job, err := s.job(account.NodeID, userID, account.ID)
	if err != nil {
		return ScheduledTask{}, jobs.Job{}, err
	}
	value, job, err = s.repository.CreateScheduledTask(ctx, value, job)
	if err == nil {
		s.notify()
	}
	return value, job, err
}

func (s *Service) Update(
	ctx context.Context, id, accountID, name, schedule, command, driver, userID string, kind Kind,
	admin, enabled bool,
) (ScheduledTask, jobs.Job, error) {
	current, err := s.get(ctx, id, userID, admin)
	if err != nil {
		return ScheduledTask{}, jobs.Job{}, err
	}
	if current.AccountID != accountID {
		return ScheduledTask{}, jobs.Job{}, accounts.ErrForbidden
	}
	if _, err := s.account(ctx, accountID, userID, admin); err != nil {
		return ScheduledTask{}, jobs.Job{}, err
	}
	current.Kind = kind
	value, err := build(current, name, schedule, command, driver, enabled)
	if err != nil {
		return ScheduledTask{}, jobs.Job{}, err
	}
	value.UpdatedAt = time.Now().UTC()
	job, err := s.job(value.NodeID, userID, value.AccountID)
	if err != nil {
		return ScheduledTask{}, jobs.Job{}, err
	}
	value, job, err = s.repository.UpdateScheduledTask(ctx, value, job)
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
	job, err = s.repository.DeleteScheduledTask(ctx, value.ID, job)
	if err == nil {
		s.notify()
	}
	return job, err
}

func (s *Service) ScheduledTasks(ctx context.Context, userID string, admin bool) ([]ScheduledTask, error) {
	return s.repository.ScheduledTasks(ctx, userID, admin)
}

func (s *Service) ScheduledTaskPage(
	ctx context.Context, userID string, admin bool, query pagination.Query,
) (pagination.Result[ScheduledTask], error) {
	return s.repository.ScheduledTaskPage(ctx, userID, admin, query)
}

func (s *Service) Sync(ctx context.Context, job jobs.Job) error {
	var payload syncPayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil || payload.AccountID == "" {
		return fmt.Errorf("decode task sync payload")
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
	values, err := s.repository.AccountScheduledTasks(ctx, account.ID)
	if err != nil {
		return err
	}
	entries := make([]Entry, 0, len(values))
	for _, value := range values {
		if value.SchedulerDriver != services.Crontab {
			return fmt.Errorf("unsupported scheduler driver %q", value.SchedulerDriver)
		}
		if value.Enabled && account.Enabled && account.Status == "active" {
			entries = append(entries, Entry{ID: value.ID, Schedule: value.Schedule, Command: value.Command, Kind: value.Kind})
		}
	}
	if err := s.agent.SyncTasks(ctx, node.Endpoint, account.ID, account.SystemUser, entries); err != nil {
		_ = s.repository.SetTaskStatuses(ctx, account.ID, "error", "error")
		return err
	}
	return s.repository.SetTaskStatuses(ctx, account.ID, "active", "disabled")
}

func (s *Service) get(ctx context.Context, id, userID string, admin bool) (ScheduledTask, error) {
	if err := validate.ID("scheduledTaskId", id); err != nil {
		return ScheduledTask{}, err
	}
	value, err := s.repository.ScheduledTask(ctx, id)
	if err != nil {
		return ScheduledTask{}, err
	}
	if _, err := s.accounts.Account(ctx, value.AccountID, userID, admin); err != nil {
		return ScheduledTask{}, err
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
	return jobs.Job{ID: id, NodeID: nodeID, UserID: userID, Kind: jobs.KindTaskSync, Status: "queued", Payload: string(payload), MaxAttempts: 2, CreatedAt: time.Now().UTC()}, nil
}

func build(
	value ScheduledTask, name, schedule, command, driver string, enabled bool,
) (ScheduledTask, error) {
	var err error
	if value.Kind != Command {
		return ScheduledTask{}, &validate.Error{Field: "kind", Message: "The selected task kind is not supported"}
	}
	value.Name, err = validate.ResourceName(name)
	if err != nil {
		return ScheduledTask{}, err
	}
	value.Schedule, err = validate.CronSchedule(schedule, false)
	if err != nil {
		return ScheduledTask{}, err
	}
	value.Command, err = validate.CronCommand(command)
	if err != nil {
		return ScheduledTask{}, err
	}
	if driver != services.Crontab {
		return ScheduledTask{}, &validate.Error{
			Field: "schedulerDriver", Message: "The selected scheduler driver is not supported",
		}
	}
	value.SchedulerDriver = driver
	value.Enabled = enabled
	value.Status = "pending"
	return value, nil
}
