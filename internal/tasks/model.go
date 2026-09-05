package tasks

import (
	"context"
	"time"

	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/pagination"
)

type ScheduledTask struct {
	ID, AccountID, NodeID, Name, Schedule, Command, SchedulerDriver, Status string
	Kind                                                                    Kind
	Enabled                                                                 bool
	CreatedAt, UpdatedAt                                                    time.Time
}

type Kind string

const Command Kind = "command"

type Repository interface {
	CreateScheduledTask(context.Context, ScheduledTask, jobs.Job) (ScheduledTask, jobs.Job, error)
	UpdateScheduledTask(context.Context, ScheduledTask, jobs.Job) (ScheduledTask, jobs.Job, error)
	DeleteScheduledTask(context.Context, string, jobs.Job) (jobs.Job, error)
	ScheduledTasks(context.Context, string, bool) ([]ScheduledTask, error)
	ScheduledTaskPage(context.Context, string, bool, pagination.Query) (pagination.Result[ScheduledTask], error)
	ScheduledTask(context.Context, string) (ScheduledTask, error)
	AccountScheduledTasks(context.Context, string) ([]ScheduledTask, error)
	SetTaskStatuses(context.Context, string, string, string) error
}

type Entry struct {
	ID, Schedule, Command string
	Kind                  Kind
}

type Agent interface {
	SyncTasks(context.Context, string, string, string, []Entry) error
}
