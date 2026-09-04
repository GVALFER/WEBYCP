package databases

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
	"github.com/GVALFER/WEBYCP/internal/secret"
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

type payload struct {
	DatabaseID string `json:"databaseId,omitempty"`
	UserID     string `json:"userId,omitempty"`
	Password   string `json:"password,omitempty"`
}

func NewService(
	repository Repository, accounts *accounts.Service, nodes nodes.Repository,
	agent Agent, notify func(),
) *Service {
	return &Service{repository: repository, accounts: accounts, nodes: nodes, agent: agent, notify: notify}
}

func (s *Service) CreateDatabase(
	ctx context.Context, accountID, name, driver, userID string, admin bool,
) (Database, jobs.Job, error) {
	account, err := s.activeAccount(ctx, accountID, userID, admin)
	if err != nil {
		return Database{}, jobs.Job{}, err
	}
	name, err = validate.DatabaseName(name)
	if err != nil {
		return Database{}, jobs.Job{}, err
	}
	if driver != services.MySQL {
		return Database{}, jobs.Job{}, &validate.Error{
			Field: "driver", Message: "The selected database driver is not supported",
		}
	}
	id, err := idgen.ID()
	if err != nil {
		return Database{}, jobs.Job{}, err
	}
	now := time.Now().UTC()
	database := Database{
		ID: id, AccountID: account.ID, NodeID: account.NodeID, Name: name,
		SystemName: "wcp_" + account.ID[:8] + "_" + name, Driver: driver, Status: "pending",
		CreatedAt: now, UpdatedAt: now,
	}
	job, err := newJob(database.NodeID, userID, jobs.KindDatabaseCreate, payload{DatabaseID: id})
	if err != nil {
		return Database{}, jobs.Job{}, err
	}
	database, job, err = s.repository.CreateDatabase(ctx, database, job)
	if err == nil {
		s.notify()
	}
	return database, job, err
}

func (s *Service) Databases(ctx context.Context, userID string, admin bool) ([]Database, error) {
	return s.repository.Databases(ctx, userID, admin)
}

func (s *Service) DatabasePage(
	ctx context.Context, userID string, admin bool, query pagination.Query,
) (pagination.Result[Database], error) {
	return s.repository.DatabasePage(ctx, userID, admin, query)
}

func (s *Service) DeleteDatabase(
	ctx context.Context, id, userID string, admin bool,
) (Database, jobs.Job, error) {
	database, err := s.database(ctx, id, userID, admin)
	if err != nil {
		return Database{}, jobs.Job{}, err
	}
	if database.Status == "pending" {
		return Database{}, jobs.Job{}, ErrBusy
	}
	job, err := newJob(database.NodeID, userID, jobs.KindDatabaseDelete, payload{DatabaseID: id})
	if err != nil {
		return Database{}, jobs.Job{}, err
	}
	database, job, err = s.repository.QueueDatabaseDelete(ctx, id, job)
	if err == nil {
		s.notify()
	}
	return database, job, err
}

func (s *Service) CreateUser(
	ctx context.Context, accountID, name, driver, userID string, admin bool,
) (User, jobs.Job, string, error) {
	account, err := s.activeAccount(ctx, accountID, userID, admin)
	if err != nil {
		return User{}, jobs.Job{}, "", err
	}
	name, err = validate.DatabaseName(name)
	if err != nil {
		return User{}, jobs.Job{}, "", err
	}
	if driver != services.MySQL {
		return User{}, jobs.Job{}, "", &validate.Error{
			Field: "driver", Message: "The selected database driver is not supported",
		}
	}
	id, err := idgen.ID()
	if err != nil {
		return User{}, jobs.Job{}, "", err
	}
	password, err := secret.Generate(24)
	if err != nil {
		return User{}, jobs.Job{}, "", err
	}
	now := time.Now().UTC()
	user := User{
		ID: id, AccountID: account.ID, NodeID: account.NodeID, Name: name,
		SystemName: "wcp_" + account.ID[:8] + "_" + name, Driver: driver, Status: "pending",
		CreatedAt: now, UpdatedAt: now,
	}
	job, err := newJob(user.NodeID, userID, jobs.KindDatabaseUserCreate, payload{
		UserID: id, Password: password,
	})
	if err != nil {
		return User{}, jobs.Job{}, "", err
	}
	user, job, err = s.repository.CreateUser(ctx, user, job)
	if err == nil {
		s.notify()
	}
	return user, job, password, err
}

func (s *Service) Users(ctx context.Context, userID string, admin bool) ([]User, error) {
	return s.repository.Users(ctx, userID, admin)
}

func (s *Service) UserPage(
	ctx context.Context, userID string, admin bool, query pagination.Query,
) (pagination.Result[User], error) {
	return s.repository.UserPage(ctx, userID, admin, query)
}

func (s *Service) DeleteUser(
	ctx context.Context, id, userID string, admin bool,
) (User, jobs.Job, error) {
	user, err := s.user(ctx, id, userID, admin)
	if err != nil {
		return User{}, jobs.Job{}, err
	}
	if user.Status == "pending" {
		return User{}, jobs.Job{}, ErrBusy
	}
	job, err := newJob(user.NodeID, userID, jobs.KindDatabaseUserDelete, payload{UserID: id})
	if err != nil {
		return User{}, jobs.Job{}, err
	}
	user, job, err = s.repository.QueueUserDelete(ctx, id, job)
	if err == nil {
		s.notify()
	}
	return user, job, err
}

func (s *Service) SetGrant(
	ctx context.Context, databaseID, databaseUserID, userID string, admin, enabled bool,
) (Grant, jobs.Job, error) {
	database, err := s.database(ctx, databaseID, userID, admin)
	if err != nil {
		return Grant{}, jobs.Job{}, err
	}
	user, err := s.user(ctx, databaseUserID, userID, admin)
	if err != nil {
		return Grant{}, jobs.Job{}, err
	}
	if database.AccountID != user.AccountID {
		return Grant{}, jobs.Job{}, ErrCrossAccount
	}
	if database.Driver != services.MySQL || user.Driver != services.MySQL {
		return Grant{}, jobs.Job{}, &validate.Error{
			Field: "driver", Message: "The selected database driver is not supported",
		}
	}
	if database.Status != "active" || user.Status != "active" {
		return Grant{}, jobs.Job{}, ErrBusy
	}
	kind := jobs.KindDatabaseGrantDelete
	if enabled {
		kind = jobs.KindDatabaseGrantCreate
	}
	job, err := newJob(database.NodeID, userID, kind, payload{DatabaseID: database.ID, UserID: user.ID})
	if err != nil {
		return Grant{}, jobs.Job{}, err
	}
	grant := Grant{DatabaseID: database.ID, UserID: user.ID, Status: "pending", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	grant, job, err = s.repository.QueueGrant(ctx, grant, enabled, job)
	if err == nil {
		s.notify()
	}
	return grant, job, err
}

func (s *Service) Grants(ctx context.Context, userID string, admin bool) ([]Grant, error) {
	return s.repository.Grants(ctx, userID, admin)
}

func (s *Service) GrantPage(
	ctx context.Context, userID string, admin bool, query pagination.Query,
) (pagination.Result[Grant], error) {
	return s.repository.GrantPage(ctx, userID, admin, query)
}

func (s *Service) Provision(ctx context.Context, job jobs.Job) error {
	var value payload
	if err := json.Unmarshal([]byte(job.Payload), &value); err != nil {
		return fmt.Errorf("decode database job payload")
	}
	node, err := s.nodes.Node(ctx, job.NodeID)
	if err != nil {
		return fmt.Errorf("get database node: %w", err)
	}
	switch job.Kind {
	case jobs.KindDatabaseCreate, jobs.KindDatabaseDelete:
		database, err := s.repository.Database(ctx, value.DatabaseID)
		if err != nil {
			return err
		}
		if database.Driver != services.MySQL {
			return fmt.Errorf("unsupported database driver %q", database.Driver)
		}
		if job.Kind == jobs.KindDatabaseCreate {
			err = s.agent.EnsureDatabase(ctx, node.Endpoint, database.SystemName)
		} else {
			err = s.agent.DeleteDatabase(ctx, node.Endpoint, database.SystemName)
		}
		if err != nil {
			_ = s.repository.SetDatabaseStatus(ctx, database.ID, "error")
			return err
		}
		if job.Kind == jobs.KindDatabaseDelete {
			return s.repository.DeleteDatabase(ctx, database.ID)
		}
		return s.repository.SetDatabaseStatus(ctx, database.ID, "active")
	case jobs.KindDatabaseUserCreate, jobs.KindDatabaseUserDelete:
		user, err := s.repository.User(ctx, value.UserID)
		if err != nil {
			return err
		}
		if user.Driver != services.MySQL {
			return fmt.Errorf("unsupported database driver %q", user.Driver)
		}
		if job.Kind == jobs.KindDatabaseUserCreate {
			err = s.agent.EnsureDatabaseUser(ctx, node.Endpoint, user.SystemName, value.Password)
		} else {
			err = s.agent.DeleteDatabaseUser(ctx, node.Endpoint, user.SystemName)
		}
		if err != nil {
			_ = s.repository.SetUserStatus(ctx, user.ID, "error")
			return err
		}
		if job.Kind == jobs.KindDatabaseUserDelete {
			return s.repository.DeleteUser(ctx, user.ID)
		}
		return s.repository.SetUserStatus(ctx, user.ID, "active")
	case jobs.KindDatabaseGrantCreate, jobs.KindDatabaseGrantDelete:
		database, err := s.repository.Database(ctx, value.DatabaseID)
		if err != nil {
			return err
		}
		user, err := s.repository.User(ctx, value.UserID)
		if err != nil {
			return err
		}
		if database.AccountID != user.AccountID {
			return ErrCrossAccount
		}
		if job.Kind == jobs.KindDatabaseGrantCreate {
			err = s.agent.EnsureDatabaseGrant(ctx, node.Endpoint, database.SystemName, user.SystemName)
		} else {
			err = s.agent.DeleteDatabaseGrant(ctx, node.Endpoint, database.SystemName, user.SystemName)
		}
		if err != nil {
			_ = s.repository.SetGrantStatus(ctx, database.ID, user.ID, "error")
			return err
		}
		if job.Kind == jobs.KindDatabaseGrantDelete {
			return s.repository.DeleteGrant(ctx, database.ID, user.ID)
		}
		return s.repository.SetGrantStatus(ctx, database.ID, user.ID, "active")
	default:
		return fmt.Errorf("unsupported database job %q", job.Kind)
	}
}

func (s *Service) activeAccount(ctx context.Context, id, userID string, admin bool) (accounts.Account, error) {
	account, err := s.accounts.Account(ctx, id, userID, admin)
	if err != nil {
		return accounts.Account{}, err
	}
	if account.Status != "active" || !account.Enabled {
		return accounts.Account{}, accounts.ErrBusy
	}
	return account, nil
}

func (s *Service) database(ctx context.Context, id, userID string, admin bool) (Database, error) {
	if err := validate.ID("databaseId", id); err != nil {
		return Database{}, err
	}
	value, err := s.repository.Database(ctx, id)
	if err != nil {
		return Database{}, err
	}
	if _, err := s.accounts.Account(ctx, value.AccountID, userID, admin); err != nil {
		return Database{}, err
	}
	return value, nil
}

func (s *Service) user(ctx context.Context, id, userID string, admin bool) (User, error) {
	if err := validate.ID("databaseUserId", id); err != nil {
		return User{}, err
	}
	value, err := s.repository.User(ctx, id)
	if err != nil {
		return User{}, err
	}
	if _, err := s.accounts.Account(ctx, value.AccountID, userID, admin); err != nil {
		return User{}, err
	}
	return value, nil
}

func newJob(nodeID, userID, kind string, value payload) (jobs.Job, error) {
	id, err := idgen.ID()
	if err != nil {
		return jobs.Job{}, err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return jobs.Job{}, err
	}
	return jobs.Job{ID: id, NodeID: nodeID, UserID: userID, Kind: kind, Status: "queued", Payload: string(data), MaxAttempts: 2, CreatedAt: time.Now().UTC()}, nil
}
