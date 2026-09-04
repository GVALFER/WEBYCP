package accounts

import (
	"context"
	"errors"
	"time"

	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/pagination"
)

var (
	ErrForbidden  = errors.New("account access denied")
	ErrNameExists = errors.New("account name already exists")
	ErrBusy       = errors.New("account operation is already pending")
	ErrNotEmpty   = errors.New("account still owns resources")
)

type Account struct {
	ID         string
	NodeID     string
	Name       string
	SystemUser string
	Status     string
	Enabled    bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Repository interface {
	CreateProvision(context.Context, Account, string, jobs.Job) (Account, jobs.Job, error)
	Account(context.Context, string) (Account, error)
	AccountMember(context.Context, string, string) (bool, error)
	Accounts(context.Context, string, bool) ([]Account, error)
	AccountPage(context.Context, string, bool, pagination.Query) (pagination.Result[Account], error)
	QueueAction(context.Context, string, bool, jobs.Job) (Account, jobs.Job, error)
	ResourceCount(context.Context, string) (int64, error)
	UpdateStatus(context.Context, string, string) error
	Delete(context.Context, string) error
}

type Agent interface {
	EnsureAccount(context.Context, string, string, string) error
	DisableAccount(context.Context, string, string, string) error
	EnableAccount(context.Context, string, string, string) error
	DeleteAccount(context.Context, string, string, string) error
}
