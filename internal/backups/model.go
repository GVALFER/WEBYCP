package backups

import (
	"context"
	"errors"
	"time"

	agentbackup "github.com/GVALFER/WEBYCP/internal/agent/backup"
	"github.com/GVALFER/WEBYCP/internal/backupfmt"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/websites"
)

var (
	ErrBusy  = errors.New("backup plan already has an active run")
	ErrScope = errors.New("backup scope is empty")
)

type Plan struct {
	ID, AccountID, NodeID, Name, Schedule, StorageDriver string
	RetentionCount                                       int64
	IncludeFiles, IncludeDatabases                       bool
	Enabled                                              bool
	LastRunAt, NextRunAt                                 *time.Time
	CreatedAt, UpdatedAt                                 time.Time
}

type Run struct {
	ID, PlanID, AccountID, NodeID, StorageDriver, Status, Error string
	CreatedAt                                                   time.Time
	StartedAt, FinishedAt                                       *time.Time
}

type Artifact struct {
	ID, RunID, AccountID, NodeID, StorageDriver, Path, Checksum string
	Size                                                        int64
	Manifest                                                    backupfmt.Manifest
	CreatedAt                                                   time.Time
}

type RestoreScope struct {
	Files, Databases, Metadata bool
}

type Repository interface {
	CreateJob(context.Context, jobs.Job) (jobs.Job, error)
	BackupPlans(context.Context, string, bool) ([]Plan, error)
	BackupPlanPage(context.Context, string, bool, pagination.Query) (pagination.Result[Plan], error)
	BackupPlan(context.Context, string) (Plan, error)
	CreateBackupPlan(context.Context, Plan) (Plan, error)
	UpdateBackupPlan(context.Context, Plan, int64) (Plan, error)
	DeleteBackupPlan(context.Context, string) error
	DueBackupPlans(context.Context, time.Time) ([]Plan, error)
	QueueBackupRun(context.Context, Run, jobs.Job, *time.Time) (Run, jobs.Job, error)
	BackupRunPending(context.Context, string) (bool, error)
	SetBackupRun(context.Context, string, string, string, *time.Time, *time.Time) error
	CompleteBackup(context.Context, Run, Artifact) (Artifact, error)
	BackupRuns(context.Context, string, bool) ([]Run, error)
	BackupRunPage(context.Context, string, bool, pagination.Query) (pagination.Result[Run], error)
	BackupArtifact(context.Context, string) (Artifact, error)
	BackupArtifacts(context.Context, string, bool) ([]Artifact, error)
	BackupArtifactPage(context.Context, string, bool, pagination.Query) (pagination.Result[Artifact], error)
	ExpiredBackupArtifacts(context.Context, string, int64) ([]Artifact, error)
	DeleteBackupArtifact(context.Context, string) error
	RestoreMetadata(context.Context, backupfmt.Metadata) error
}

type Agent interface {
	CreateBackup(context.Context, string, agentbackup.CreateRequest) (agentbackup.Artifact, error)
	PreviewBackup(context.Context, string, agentbackup.ArtifactRequest) (backupfmt.Manifest, error)
	RestoreBackup(context.Context, string, agentbackup.RestoreRequest) (string, error)
	DeleteBackup(context.Context, string, agentbackup.ArtifactRequest) error
	EnsureWebsite(context.Context, string, websites.Spec) error
}

type CertificateReconciler interface {
	ReconcileWebsite(context.Context, string) error
}
