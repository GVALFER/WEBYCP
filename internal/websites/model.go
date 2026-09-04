package websites

import (
	"context"
	"errors"
	"time"

	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/pagination"
)

var (
	ErrWebsiteBusy       = errors.New("website operation is pending")
	ErrWebsiteDomainBusy = errors.New("website domain operation is pending")
	ErrWebsiteInactive   = errors.New("website is not active")
	ErrHostnameExists    = errors.New("hostname already exists")
	ErrHostnameSame      = errors.New("hostname is unchanged")
	ErrPrimaryRequired   = errors.New("the primary domain cannot be deleted or disabled")
)

type Website struct {
	ID, AccountID, NodeID                    string
	Name, Kind, DocumentRoot                 string
	WebDriver, RuntimeDriver, RuntimeVersion string
	Status                                   string
	Enabled                                  bool
	CreatedAt, UpdatedAt                     time.Time
}

type WebsiteDomain struct {
	ID, WebsiteID, Hostname, Kind, Status, PreviousHostname string
	Enabled                                                 bool
	CreatedAt, UpdatedAt                                    time.Time
}

type Spec struct {
	AccountID, SystemUser, WebsiteID, DocumentRoot string
	Kind, WebDriver, RuntimeDriver, RuntimeVersion string
	PrimaryDomain                                  string
	Aliases                                        []string
}

type Repository interface {
	CreateWebsiteProvision(context.Context, Website, WebsiteDomain, jobs.Job) (Website, WebsiteDomain, jobs.Job, error)
	CreateWebsiteDomainProvision(context.Context, WebsiteDomain, jobs.Job) (WebsiteDomain, jobs.Job, error)
	Website(context.Context, string) (Website, error)
	Websites(context.Context, string, bool) ([]Website, error)
	WebsitePage(context.Context, string, bool, pagination.Query) (pagination.Result[Website], error)
	WebsiteDomain(context.Context, string) (WebsiteDomain, error)
	PrimaryDomain(context.Context, string) (WebsiteDomain, error)
	WebsiteDomains(context.Context, string) ([]WebsiteDomain, error)
	EnabledWebsiteDomains(context.Context, string) ([]WebsiteDomain, error)
	WebsiteDomainPage(context.Context, string, bool, string, pagination.Query) (pagination.Result[WebsiteDomain], error)
	QueueWebsiteAction(context.Context, string, bool, jobs.Job) (Website, jobs.Job, error)
	QueueWebsiteDomainAction(context.Context, string, bool, jobs.Job) (WebsiteDomain, jobs.Job, error)
	QueueWebsiteDomainRename(context.Context, string, string, jobs.Job) (WebsiteDomain, jobs.Job, error)
	UpdateWebsiteStatus(context.Context, string, string) error
	UpdateWebsiteDomainStatus(context.Context, string, string) error
	CompleteWebsiteDomainRename(context.Context, string) error
	FailWebsiteDomainRename(context.Context, string) error
	DeleteWebsite(context.Context, string) error
	DeleteWebsiteDomain(context.Context, string) error
}

type Agent interface {
	EnsureWebsite(context.Context, string, Spec) error
	DisableWebsite(context.Context, string, Spec) error
	DeleteWebsite(context.Context, string, Spec) error
}
