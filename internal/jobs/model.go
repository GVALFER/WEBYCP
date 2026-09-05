package jobs

import (
	"context"
	"errors"
	"time"

	"github.com/GVALFER/WEBYCP/internal/pagination"
)

const (
	KindAccountCreate        = "account.create"
	KindAccountDelete        = "account.delete"
	KindAccountDisable       = "account.disable"
	KindAccountEnable        = "account.enable"
	KindWebsiteCreate        = "website.create"
	KindWebsiteDelete        = "website.delete"
	KindWebsiteDisable       = "website.disable"
	KindWebsiteEnable        = "website.enable"
	KindWebsiteDomainCreate  = "website_domain.create"
	KindWebsiteDomainDelete  = "website_domain.delete"
	KindWebsiteDomainDisable = "website_domain.disable"
	KindWebsiteDomainEnable  = "website_domain.enable"
	KindWebsiteDomainUpdate  = "website_domain.update"
	KindNodeProbe            = "node.probe"
	KindCertificateIssue     = "certificate.issue"
	KindCertificateRenew     = "certificate.renew"
	KindDatabaseCreate       = "database.create"
	KindDatabaseDelete       = "database.delete"
	KindDatabaseUserCreate   = "database_user.create"
	KindDatabaseUserDelete   = "database_user.delete"
	KindDatabaseGrantCreate  = "database_grant.create"
	KindDatabaseGrantDelete  = "database_grant.delete"
	KindTaskSync             = "task.sync"
	KindBackupCreate         = "backup.create"
	KindBackupRestore        = "backup.restore"
	KindDNSZoneCreate        = "dns_zone.create"
	KindDNSZoneDelete        = "dns_zone.delete"
	KindDNSRecordSync        = "dns_record.sync"
)

var ErrNone = errors.New("no queued jobs")

type Job struct {
	ID          string
	NodeID      string
	UserID      string
	Kind        string
	Status      string
	Payload     string
	Attempts    int64
	MaxAttempts int64
	Error       string
	CreatedAt   time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
}

type Step struct {
	ID         string
	JobID      string
	Name       string
	Status     string
	Message    string
	StartedAt  time.Time
	FinishedAt *time.Time
}

type Repository interface {
	CreateJob(context.Context, Job) (Job, error)
	Job(context.Context, string) (Job, error)
	Jobs(context.Context, int64) ([]Job, error)
	JobPage(context.Context, pagination.Query) (pagination.Result[Job], error)
	ClaimJob(context.Context, time.Time) (Job, error)
	CompleteJob(context.Context, string, time.Time) error
	RetryJob(context.Context, string, string) error
	FailJob(context.Context, string, string, time.Time) error
	RecoverJobs(context.Context, time.Time) error
	CreateStep(context.Context, Step) (Step, error)
	FinishStep(context.Context, string, string, string, time.Time) error
	Steps(context.Context, string) ([]Step, error)
}
