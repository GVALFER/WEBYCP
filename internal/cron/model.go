package cronjob

import (
	"context"
	"time"

	"github.com/GVALFER/WEBYCP/internal/jobs"
)

type CronJob struct {
	ID, AccountID, NodeID, Name, Schedule, Command, Status string
	Enabled                                                bool
	CreatedAt, UpdatedAt                                   time.Time
}

type Repository interface {
	CreateCronJob(context.Context, CronJob, jobs.Job) (CronJob, jobs.Job, error)
	UpdateCronJob(context.Context, CronJob, jobs.Job) (CronJob, jobs.Job, error)
	DeleteCronJob(context.Context, string, jobs.Job) (jobs.Job, error)
	CronJobs(context.Context, string, bool) ([]CronJob, error)
	CronJob(context.Context, string) (CronJob, error)
	AccountCronJobs(context.Context, string) ([]CronJob, error)
	SetCronStatuses(context.Context, string, string, string) error
}

type Entry struct {
	ID, Schedule, Command string
}

type Agent interface {
	SyncCron(context.Context, string, string, string, []Entry) error
}
