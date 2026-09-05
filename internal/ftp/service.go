package ftp

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	agentftp "github.com/GVALFER/WEBYCP/internal/agent/ftp"
	"github.com/GVALFER/WEBYCP/internal/auth"
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

func (s *Service) Create(ctx context.Context, accountID, username, password, userID string, admin, enabled bool) (Account, jobs.Job, error) {
	account, err := s.accounts.Account(ctx, accountID, userID, admin)
	if err != nil {
		return Account{}, jobs.Job{}, err
	}
	if account.Status != "active" || !account.Enabled {
		return Account{}, jobs.Job{}, accounts.ErrBusy
	}
	username, err = validate.Username(username)
	if err != nil {
		return Account{}, jobs.Job{}, err
	}
	hash, err := passwordHash(password)
	if err != nil {
		return Account{}, jobs.Job{}, err
	}
	id, err := idgen.ID()
	if err != nil {
		return Account{}, jobs.Job{}, err
	}
	job, err := syncJob(account.NodeID, userID, account.ID)
	if err != nil {
		return Account{}, jobs.Job{}, err
	}
	now := time.Now().UTC()
	value, job, err := s.repository.CreateFTP(ctx, Credential{
		Account: Account{ID: id, AccountID: account.ID, NodeID: account.NodeID, Username: username,
			Enabled: enabled, Status: "pending", CreatedAt: now, UpdatedAt: now},
		PasswordHash: hash,
	}, job)
	if err == nil {
		s.notify()
	}
	return value, job, err
}

func (s *Service) Update(ctx context.Context, id, userID string, admin bool, username, password *string, enabled *bool) (Account, jobs.Job, error) {
	current, err := s.get(ctx, id, userID, admin)
	if err != nil {
		return Account{}, jobs.Job{}, err
	}
	if username == nil && password == nil && enabled == nil {
		return Account{}, jobs.Job{}, &validate.Error{Field: "body", Message: "Provide a username, password or enabled state"}
	}
	change := Changes{Enabled: enabled}
	if username != nil {
		name, err := validate.Username(*username)
		if err != nil {
			return Account{}, jobs.Job{}, err
		}
		change.Username = &name
	}
	if password != nil {
		hash, err := passwordHash(*password)
		if err != nil {
			return Account{}, jobs.Job{}, err
		}
		change.PasswordHash = &hash
	}
	return s.change(ctx, current.Account, userID, change)
}

func (s *Service) Delete(ctx context.Context, id, userID string, admin bool) (jobs.Job, error) {
	current, err := s.get(ctx, id, userID, admin)
	if err != nil {
		return jobs.Job{}, err
	}
	_, job, err := s.change(ctx, current.Account, userID, Changes{Deleting: true})
	return job, err
}

func (s *Service) Page(ctx context.Context, userID string, admin bool, query pagination.Query) (pagination.Result[Account], error) {
	return s.repository.FTPPage(ctx, userID, admin, query)
}

func (s *Service) Sync(ctx context.Context, job jobs.Job) error {
	var payload syncPayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil || validate.ID("accountId", payload.AccountID) != nil {
		return errors.New("invalid FTP sync payload")
	}
	err := s.reconcile(ctx, payload.AccountID)
	if err != nil {
		_ = s.repository.FinishFTP(ctx, payload.AccountID, true)
		// Agent transport errors may contain response details; never put them in a public Job.
		return errors.New("FTP synchronization failed; check the Agent and retry the change")
	}
	return s.repository.FinishFTP(ctx, payload.AccountID, false)
}

func (s *Service) reconcile(ctx context.Context, accountID string) error {
	account, err := s.accounts.Get(ctx, accountID)
	if err != nil {
		return err
	}
	node, err := s.nodes.Node(ctx, account.NodeID)
	if err != nil {
		return err
	}
	values, err := s.repository.AccountFTP(ctx, accountID)
	if err != nil {
		return err
	}
	entries := make([]agentftp.Entry, 0, len(values))
	for _, value := range values {
		if !value.Deleting {
			entries = append(entries, agentftp.Entry{ID: value.ID, Username: value.Username,
				PasswordHash: value.PasswordHash, Enabled: value.Enabled})
		}
	}
	return s.agent.SyncFTP(ctx, node.Endpoint, accountID, account.SystemUser, entries)
}

func (s *Service) get(ctx context.Context, id, userID string, admin bool) (Credential, error) {
	if err := validate.ID("ftpAccountId", id); err != nil {
		return Credential{}, err
	}
	value, err := s.repository.FTP(ctx, id)
	if err != nil {
		return Credential{}, err
	}
	if _, err := s.accounts.Account(ctx, value.AccountID, userID, admin); err != nil {
		return Credential{}, err
	}
	return value, nil
}

func (s *Service) change(ctx context.Context, value Account, userID string, change Changes) (Account, jobs.Job, error) {
	job, err := syncJob(value.NodeID, userID, value.AccountID)
	if err != nil {
		return Account{}, jobs.Job{}, err
	}
	value, job, err = s.repository.ChangeFTP(ctx, value.ID, change, job)
	if err == nil {
		s.notify()
	}
	return value, job, err
}

func syncJob(nodeID, userID, accountID string) (jobs.Job, error) {
	id, err := idgen.ID()
	if err != nil {
		return jobs.Job{}, err
	}
	payload, err := json.Marshal(syncPayload{AccountID: accountID})
	if err != nil {
		return jobs.Job{}, err
	}
	return jobs.Job{ID: id, NodeID: nodeID, UserID: userID, Kind: jobs.KindFTPSync,
		Status: "queued", Payload: string(payload), MaxAttempts: 2, CreatedAt: time.Now().UTC()}, nil
}

func passwordHash(password string) (string, error) {
	if err := validate.Password(password); err != nil {
		return "", err
	}
	return auth.HashPassword(password)
}
