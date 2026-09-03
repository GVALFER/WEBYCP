package databases

import (
	"context"
	"errors"
	"time"

	"github.com/GVALFER/WEBYCP/internal/jobs"
)

var (
	ErrBusy         = errors.New("database resource operation is pending")
	ErrNameExists   = errors.New("database resource name exists")
	ErrCrossAccount = errors.New("database grant crosses account boundary")
)

type Database struct {
	ID, AccountID, NodeID, Name, SystemName, Status string
	CreatedAt, UpdatedAt                            time.Time
}

type User struct {
	ID, AccountID, NodeID, Name, SystemName, Status string
	CreatedAt, UpdatedAt                            time.Time
}

type Grant struct {
	DatabaseID, UserID, Status string
	CreatedAt, UpdatedAt       time.Time
}

type Repository interface {
	CreateDatabase(context.Context, Database, jobs.Job) (Database, jobs.Job, error)
	Databases(context.Context, string, bool) ([]Database, error)
	Database(context.Context, string) (Database, error)
	QueueDatabaseDelete(context.Context, string, jobs.Job) (Database, jobs.Job, error)
	SetDatabaseStatus(context.Context, string, string) error
	DeleteDatabase(context.Context, string) error
	CreateUser(context.Context, User, jobs.Job) (User, jobs.Job, error)
	Users(context.Context, string, bool) ([]User, error)
	User(context.Context, string) (User, error)
	QueueUserDelete(context.Context, string, jobs.Job) (User, jobs.Job, error)
	SetUserStatus(context.Context, string, string) error
	DeleteUser(context.Context, string) error
	QueueGrant(context.Context, Grant, bool, jobs.Job) (Grant, jobs.Job, error)
	Grants(context.Context, string, bool) ([]Grant, error)
	Grant(context.Context, string, string) (Grant, error)
	SetGrantStatus(context.Context, string, string, string) error
	DeleteGrant(context.Context, string, string) error
}

type Agent interface {
	EnsureDatabase(context.Context, string, string) error
	DeleteDatabase(context.Context, string, string) error
	EnsureDatabaseUser(context.Context, string, string, string) error
	DeleteDatabaseUser(context.Context, string, string) error
	EnsureDatabaseGrant(context.Context, string, string, string) error
	DeleteDatabaseGrant(context.Context, string, string, string) error
}
