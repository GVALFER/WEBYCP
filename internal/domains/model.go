package domains

import (
	"context"
	"errors"
	"time"

	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/pagination"
)

var (
	ErrAccountInactive = errors.New("account is not active")
	ErrDomainInactive  = errors.New("domain is not active")
	ErrBusy            = errors.New("resource operation is already pending")
	ErrAliasNotFound   = errors.New("domain alias not found")
	ErrNameUnchanged   = errors.New("domain name is unchanged")
	ErrNameExists      = errors.New("domain name already exists")
)

type Domain struct {
	ID           string
	AccountID    string
	NodeID       string
	Name         string
	Status       string
	PHPVersion   string
	Enabled      bool
	PreviousName string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Alias struct {
	ID           string
	DomainID     string
	Name         string
	Status       string
	Enabled      bool
	PreviousName string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Repository interface {
	CreateDomainProvision(context.Context, Domain, jobs.Job) (Domain, jobs.Job, error)
	CreateAliasProvision(context.Context, Alias, jobs.Job) (Alias, jobs.Job, error)
	Domain(context.Context, string) (Domain, error)
	Domains(context.Context, string, bool) ([]Domain, error)
	DomainPage(context.Context, string, bool, pagination.Query) (pagination.Result[Domain], error)
	Alias(context.Context, string) (Alias, error)
	Aliases(context.Context, string) ([]Alias, error)
	AliasPage(context.Context, string, pagination.Query) (pagination.Result[Alias], error)
	EnabledAliases(context.Context, string) ([]Alias, error)
	QueueDomainAction(context.Context, string, bool, jobs.Job) (Domain, jobs.Job, error)
	QueueAliasAction(context.Context, string, bool, jobs.Job) (Alias, jobs.Job, error)
	QueueDomainRename(context.Context, string, string, jobs.Job) (Domain, jobs.Job, error)
	QueueAliasRename(context.Context, string, string, jobs.Job) (Alias, jobs.Job, error)
	UpdateDomainStatus(context.Context, string, string) error
	UpdateAliasStatus(context.Context, string, string) error
	DeleteDomain(context.Context, string) error
	DeleteAlias(context.Context, string) error
	CompleteDomainRename(context.Context, string) error
	CompleteAliasRename(context.Context, string) error
	FailDomainRename(context.Context, string) error
	FailAliasRename(context.Context, string) error
}

type Agent interface {
	EnsureDomain(context.Context, string, string, string, string, string, string, []string) error
	DisableDomain(context.Context, string, string, string, string) error
	DeleteDomain(context.Context, string, string, string, string, string) error
	RenameDomain(context.Context, string, string, string, string, string, string, string, []string) error
}
