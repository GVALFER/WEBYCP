package backups

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	agentbackup "github.com/GVALFER/WEBYCP/internal/agent/backup"
	"github.com/GVALFER/WEBYCP/internal/backupfmt"
	cronjob "github.com/GVALFER/WEBYCP/internal/cron"
	"github.com/GVALFER/WEBYCP/internal/databases"
	"github.com/GVALFER/WEBYCP/internal/idgen"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/nodes"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/services"
	"github.com/GVALFER/WEBYCP/internal/validate"
	"github.com/GVALFER/WEBYCP/internal/websites"
	robfigcron "github.com/robfig/cron/v3"
)

type Service struct {
	repository Repository
	accounts   *accounts.Service
	websites   *websites.Service
	databases  *databases.Service
	cron       *cronjob.Service
	certs      CertificateReconciler
	nodes      nodes.Repository
	agent      Agent
	notify     func()
}

type runPayload struct {
	RunID string `json:"runId"`
}

type restorePayload struct {
	ArtifactID string       `json:"artifactId"`
	Scope      RestoreScope `json:"scope"`
}

func NewService(repository Repository, accounts *accounts.Service, websites *websites.Service, databases *databases.Service, cron *cronjob.Service, certs CertificateReconciler, nodes nodes.Repository, agent Agent, notify func()) *Service {
	return &Service{repository: repository, accounts: accounts, websites: websites, databases: databases, cron: cron, certs: certs, nodes: nodes, agent: agent, notify: notify}
}

func (s *Service) Plans(ctx context.Context, userID string, admin bool) ([]Plan, error) {
	return s.repository.BackupPlans(ctx, userID, admin)
}

func (s *Service) PlanPage(
	ctx context.Context, userID string, admin bool, query pagination.Query,
) (pagination.Result[Plan], error) {
	return s.repository.BackupPlanPage(ctx, userID, admin, query)
}

func (s *Service) CreatePlan(ctx context.Context, value Plan, userID string, admin bool) (Plan, error) {
	account, err := s.activeAccount(ctx, value.AccountID, userID, admin)
	if err != nil {
		return Plan{}, err
	}
	value.ID, err = idgen.ID()
	if err != nil {
		return Plan{}, err
	}
	value.NodeID = account.NodeID
	if err := preparePlan(&value); err != nil {
		return Plan{}, err
	}
	now := time.Now().UTC()
	value.CreatedAt, value.UpdatedAt = now, now
	value.NextRunAt = nextRun(value.Schedule, value.Enabled, now)
	return s.repository.CreateBackupPlan(ctx, value)
}

func (s *Service) UpdatePlan(ctx context.Context, id string, value Plan, userID string, admin bool) (Plan, error) {
	current, err := s.plan(ctx, id, userID, admin)
	if err != nil {
		return Plan{}, err
	}
	if current.AccountID != value.AccountID {
		return Plan{}, accounts.ErrForbidden
	}
	value.ID, value.NodeID, value.CreatedAt = current.ID, current.NodeID, current.CreatedAt
	value.LastRunAt = current.LastRunAt
	if err := preparePlan(&value); err != nil {
		return Plan{}, err
	}
	value.UpdatedAt = time.Now().UTC()
	value.NextRunAt = nextRun(value.Schedule, value.Enabled, value.UpdatedAt)
	return s.repository.UpdateBackupPlan(ctx, value, current.RetentionCount)
}

func (s *Service) DeletePlan(ctx context.Context, id, userID string, admin bool) error {
	value, err := s.plan(ctx, id, userID, admin)
	if err != nil {
		return err
	}
	pending, err := s.repository.BackupRunPending(ctx, value.ID)
	if err != nil {
		return err
	}
	if pending {
		return ErrBusy
	}
	return s.repository.DeleteBackupPlan(ctx, value.ID)
}

func (s *Service) Run(ctx context.Context, id, userID string, admin bool) (Run, jobs.Job, error) {
	plan, err := s.plan(ctx, id, userID, admin)
	if err != nil {
		return Run{}, jobs.Job{}, err
	}
	if _, err := s.activeAccount(ctx, plan.AccountID, userID, admin); err != nil {
		return Run{}, jobs.Job{}, err
	}
	return s.queue(ctx, plan, userID)
}

func (s *Service) QueueDue(ctx context.Context) error {
	plans, err := s.repository.DueBackupPlans(ctx, time.Now().UTC())
	if err != nil {
		return err
	}
	for _, plan := range plans {
		if _, _, err := s.queue(ctx, plan, ""); err != nil && !errors.Is(err, ErrBusy) {
			return err
		}
	}
	return nil
}

func (s *Service) Runs(ctx context.Context, userID string, admin bool) ([]Run, error) {
	return s.repository.BackupRuns(ctx, userID, admin)
}

func (s *Service) RunPage(
	ctx context.Context, userID string, admin bool, query pagination.Query,
) (pagination.Result[Run], error) {
	return s.repository.BackupRunPage(ctx, userID, admin, query)
}

func (s *Service) Artifacts(ctx context.Context, userID string, admin bool) ([]Artifact, error) {
	return s.repository.BackupArtifacts(ctx, userID, admin)
}

func (s *Service) ArtifactPage(
	ctx context.Context, userID string, admin bool, query pagination.Query,
) (pagination.Result[Artifact], error) {
	return s.repository.BackupArtifactPage(ctx, userID, admin, query)
}

func (s *Service) DeleteArtifact(ctx context.Context, id, userID string, admin bool) error {
	artifact, _, node, err := s.artifact(ctx, id, userID, admin)
	if err != nil {
		return err
	}
	if artifact.StorageDriver != services.Local {
		return fmt.Errorf("unsupported backup driver %q", artifact.StorageDriver)
	}
	request := agentbackup.ArtifactRequest{
		AccountID: artifact.AccountID, Path: artifact.Path, Checksum: artifact.Checksum,
	}
	if err := s.agent.DeleteBackup(ctx, node.Endpoint, request); err != nil {
		return err
	}
	return s.repository.DeleteBackupArtifact(ctx, artifact.ID)
}

func (s *Service) Preview(ctx context.Context, id, userID string, admin bool) (backupfmt.Manifest, error) {
	artifact, account, node, err := s.artifact(ctx, id, userID, admin)
	if err != nil {
		return backupfmt.Manifest{}, err
	}
	if artifact.StorageDriver != services.Local {
		return backupfmt.Manifest{}, fmt.Errorf("unsupported backup driver %q", artifact.StorageDriver)
	}
	_ = account
	return s.agent.PreviewBackup(ctx, node.Endpoint, agentbackup.ArtifactRequest{AccountID: artifact.AccountID, Path: artifact.Path, Checksum: artifact.Checksum})
}

func (s *Service) Restore(ctx context.Context, id, userID string, admin bool, scope RestoreScope) (jobs.Job, error) {
	artifact, _, _, err := s.artifact(ctx, id, userID, admin)
	if err != nil {
		return jobs.Job{}, err
	}
	if err := artifact.Manifest.ValidateScope(scope.Files, scope.Databases, scope.Metadata); err != nil {
		return jobs.Job{}, err
	}
	jobID, err := idgen.ID()
	if err != nil {
		return jobs.Job{}, err
	}
	data, err := json.Marshal(restorePayload{ArtifactID: artifact.ID, Scope: scope})
	if err != nil {
		return jobs.Job{}, err
	}
	job := jobs.Job{ID: jobID, NodeID: artifact.NodeID, UserID: userID, Kind: jobs.KindBackupRestore, Status: "queued", Payload: string(data), MaxAttempts: 1, CreatedAt: time.Now().UTC()}
	job, err = s.repositoryJob(ctx, job)
	if err == nil {
		s.notify()
	}
	return job, err
}

func (s *Service) Create(ctx context.Context, job jobs.Job) error {
	var payload runPayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil || payload.RunID == "" {
		return fmt.Errorf("decode backup job payload")
	}
	run, err := s.run(ctx, payload.RunID)
	if err != nil {
		return err
	}
	if run.StorageDriver != services.Local {
		return s.failRun(ctx, run.ID, fmt.Errorf("unsupported backup driver %q", run.StorageDriver))
	}
	started := time.Now().UTC()
	if err := s.repository.SetBackupRun(ctx, run.ID, "running", "", &started, nil); err != nil {
		return err
	}
	plan, err := s.repository.BackupPlan(ctx, run.PlanID)
	if err != nil {
		return s.failRun(ctx, run.ID, err)
	}
	account, err := s.accounts.Get(ctx, run.AccountID)
	if err != nil {
		return s.failRun(ctx, run.ID, err)
	}
	node, err := s.nodes.Node(ctx, run.NodeID)
	if err != nil {
		return s.failRun(ctx, run.ID, err)
	}
	metadata, names, err := s.metadata(ctx, account.ID, plan.IncludeDatabases)
	if err != nil {
		return s.failRun(ctx, run.ID, err)
	}
	result, err := s.agent.CreateBackup(ctx, node.Endpoint, agentbackup.CreateRequest{RunID: run.ID, AccountID: account.ID, SystemUser: account.SystemUser, IncludeFiles: plan.IncludeFiles, Databases: names, Metadata: metadata})
	if err != nil {
		return s.failRun(ctx, run.ID, err)
	}
	artifactID, err := idgen.ID()
	if err != nil {
		return s.failRun(ctx, run.ID, err)
	}
	artifact := Artifact{ID: artifactID, RunID: run.ID, AccountID: run.AccountID, NodeID: run.NodeID, StorageDriver: run.StorageDriver, Path: result.Path, Checksum: result.Checksum, Size: result.Size, Manifest: result.Manifest, CreatedAt: time.Now().UTC()}
	if _, err := s.repository.CompleteBackup(ctx, run, artifact); err != nil {
		return s.failRun(ctx, run.ID, err)
	}
	return s.applyRetention(ctx, plan, node.Endpoint)
}

func (s *Service) RestoreJob(ctx context.Context, job jobs.Job) error {
	var payload restorePayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil || payload.ArtifactID == "" {
		return fmt.Errorf("decode restore job payload")
	}
	artifact, err := s.repository.BackupArtifact(ctx, payload.ArtifactID)
	if err != nil {
		return err
	}
	if artifact.StorageDriver != services.Local {
		return fmt.Errorf("unsupported backup driver %q", artifact.StorageDriver)
	}
	account, err := s.accounts.Get(ctx, artifact.AccountID)
	if err != nil {
		return err
	}
	node, err := s.nodes.Node(ctx, artifact.NodeID)
	if err != nil {
		return err
	}
	metadata, err := s.agent.RestoreBackup(ctx, node.Endpoint, agentbackup.RestoreRequest{ArtifactRequest: agentbackup.ArtifactRequest{AccountID: artifact.AccountID, Path: artifact.Path, Checksum: artifact.Checksum}, SystemUser: account.SystemUser, Files: payload.Scope.Files, Databases: payload.Scope.Databases, Metadata: payload.Scope.Metadata})
	if err != nil {
		return err
	}
	if payload.Scope.Metadata {
		var value backupfmt.Metadata
		if err := json.Unmarshal([]byte(metadata), &value); err != nil {
			return fmt.Errorf("decode restored metadata: %w", err)
		}
		if err := s.repository.RestoreMetadata(ctx, value); err != nil {
			return err
		}
		if err := s.cron.Reconcile(ctx, account.ID); err != nil {
			return fmt.Errorf("reconcile restored cron: %w", err)
		}
		for _, website := range value.Websites {
			if !website.Enabled {
				continue
			}
			spec := websites.Spec{AccountID: account.ID, SystemUser: account.SystemUser, WebsiteID: website.ID, DocumentRoot: website.DocumentRoot, Kind: website.Kind, WebDriver: website.WebDriver, RuntimeDriver: website.RuntimeDriver, RuntimeVersion: website.RuntimeVersion}
			for _, domain := range website.Domains {
				if !domain.Enabled {
					continue
				}
				if domain.Kind == "primary" {
					spec.PrimaryDomain = domain.Hostname
				} else {
					spec.Aliases = append(spec.Aliases, domain.Hostname)
				}
			}
			if err := s.agent.EnsureWebsite(ctx, node.Endpoint, spec); err != nil {
				return fmt.Errorf("reconcile restored website: %w", err)
			}
			if err := s.certs.ReconcileWebsite(ctx, website.ID); err != nil {
				return fmt.Errorf("reconcile restored certificate: %w", err)
			}
		}
	}
	return nil
}

func (s *Service) queue(ctx context.Context, plan Plan, userID string) (Run, jobs.Job, error) {
	pending, err := s.repository.BackupRunPending(ctx, plan.ID)
	if err != nil {
		return Run{}, jobs.Job{}, err
	}
	if pending {
		return Run{}, jobs.Job{}, ErrBusy
	}
	runID, err := idgen.ID()
	if err != nil {
		return Run{}, jobs.Job{}, err
	}
	jobID, err := idgen.ID()
	if err != nil {
		return Run{}, jobs.Job{}, err
	}
	data, _ := json.Marshal(runPayload{RunID: runID})
	now := time.Now().UTC()
	run := Run{ID: runID, PlanID: plan.ID, AccountID: plan.AccountID, NodeID: plan.NodeID, StorageDriver: plan.StorageDriver, Status: "queued", CreatedAt: now}
	job := jobs.Job{ID: jobID, NodeID: plan.NodeID, UserID: userID, Kind: jobs.KindBackupCreate, Status: "queued", Payload: string(data), MaxAttempts: 1, CreatedAt: now}
	next := nextRun(plan.Schedule, plan.Enabled, now)
	run, job, err = s.repository.QueueBackupRun(ctx, run, job, next)
	if err == nil {
		s.notify()
	}
	return run, job, err
}

func (s *Service) metadata(ctx context.Context, accountID string, includeDatabases bool) (string, []string, error) {
	value := backupfmt.Metadata{Version: backupfmt.Version, AccountID: accountID, Websites: []backupfmt.Website{}, Databases: []backupfmt.Database{}, CronJobs: []backupfmt.CronJob{}}
	websiteValues, err := s.websites.Websites(ctx, "", true)
	if err != nil {
		return "", nil, err
	}
	for _, website := range websiteValues {
		if website.AccountID != accountID {
			continue
		}
		item := backupfmt.Website{ID: website.ID, NodeID: website.NodeID, Name: website.Name, Kind: website.Kind, DocumentRoot: website.DocumentRoot, WebDriver: website.WebDriver, RuntimeDriver: website.RuntimeDriver, RuntimeVersion: website.RuntimeVersion, Enabled: website.Enabled, Domains: []backupfmt.WebsiteDomain{}}
		domains, err := s.websites.Domains(ctx, website.ID, "", true)
		if err != nil {
			return "", nil, err
		}
		for _, domain := range domains {
			item.Domains = append(item.Domains, backupfmt.WebsiteDomain{ID: domain.ID, Hostname: domain.Hostname, Kind: domain.Kind, Enabled: domain.Enabled})
		}
		value.Websites = append(value.Websites, item)
	}
	databaseValues, err := s.databases.Databases(ctx, "", true)
	if err != nil {
		return "", nil, err
	}
	names := []string{}
	for _, database := range databaseValues {
		if database.AccountID == accountID && database.Status == "active" {
			value.Databases = append(value.Databases, backupfmt.Database{ID: database.ID, NodeID: database.NodeID, Name: database.Name, SystemName: database.SystemName, Driver: database.Driver})
			if includeDatabases {
				names = append(names, database.SystemName)
			}
		}
	}
	cronValues, err := s.cron.CronJobs(ctx, "", true)
	if err != nil {
		return "", nil, err
	}
	for _, item := range cronValues {
		if item.AccountID == accountID {
			value.CronJobs = append(value.CronJobs, backupfmt.CronJob{ID: item.ID, NodeID: item.NodeID, Name: item.Name, Schedule: item.Schedule, Command: item.Command, SchedulerDriver: item.SchedulerDriver, Enabled: item.Enabled})
		}
	}
	data, err := json.Marshal(value)
	return string(data), names, err
}

func (s *Service) applyRetention(ctx context.Context, plan Plan, socket string) error {
	items, err := s.repository.ExpiredBackupArtifacts(ctx, plan.ID, plan.RetentionCount)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.StorageDriver != services.Local {
			return fmt.Errorf("unsupported backup driver %q", item.StorageDriver)
		}
		request := agentbackup.ArtifactRequest{AccountID: item.AccountID, Path: item.Path, Checksum: item.Checksum}
		if err := s.agent.DeleteBackup(ctx, socket, request); err != nil {
			return err
		}
		if err := s.repository.DeleteBackupArtifact(ctx, item.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) artifact(ctx context.Context, id, userID string, admin bool) (Artifact, accounts.Account, nodes.Node, error) {
	if err := validate.ID("backupArtifactId", id); err != nil {
		return Artifact{}, accounts.Account{}, nodes.Node{}, err
	}
	artifact, err := s.repository.BackupArtifact(ctx, id)
	if err != nil {
		return Artifact{}, accounts.Account{}, nodes.Node{}, err
	}
	account, err := s.accounts.Account(ctx, artifact.AccountID, userID, admin)
	if err != nil {
		return Artifact{}, accounts.Account{}, nodes.Node{}, err
	}
	node, err := s.nodes.Node(ctx, artifact.NodeID)
	return artifact, account, node, err
}

func (s *Service) plan(ctx context.Context, id, userID string, admin bool) (Plan, error) {
	if err := validate.ID("backupPlanId", id); err != nil {
		return Plan{}, err
	}
	plan, err := s.repository.BackupPlan(ctx, id)
	if err != nil {
		return Plan{}, err
	}
	if _, err := s.accounts.Account(ctx, plan.AccountID, userID, admin); err != nil {
		return Plan{}, err
	}
	return plan, nil
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

func preparePlan(value *Plan) error {
	var err error
	value.Name, err = validate.ResourceName(value.Name)
	if err != nil {
		return err
	}
	value.Schedule, err = validate.CronSchedule(value.Schedule, true)
	if err != nil {
		return err
	}
	if value.RetentionCount < 1 || value.RetentionCount > 100 {
		return &validate.Error{Field: "retentionCount", Message: "Use a retention between 1 and 100"}
	}
	if value.StorageDriver != services.Local {
		return &validate.Error{
			Field: "storageDriver", Message: "The selected backup driver is not supported",
		}
	}
	if !value.IncludeFiles && !value.IncludeDatabases {
		return ErrScope
	}
	return nil
}

func nextRun(schedule string, enabled bool, now time.Time) *time.Time {
	if !enabled || schedule == "" {
		return nil
	}
	parsed, err := robfigcron.ParseStandard(schedule)
	if err != nil {
		return nil
	}
	next := parsed.Next(now).UTC()
	return &next
}

func (s *Service) failRun(ctx context.Context, id string, cause error) error {
	now := time.Now().UTC()
	_ = s.repository.SetBackupRun(ctx, id, "failed", safeMessage(cause), nil, &now)
	return cause
}

func safeMessage(err error) string {
	value := err.Error()
	if len(value) > 500 {
		return value[:500]
	}
	return value
}

func (s *Service) run(ctx context.Context, id string) (Run, error) {
	runs, err := s.repository.BackupRuns(ctx, "", true)
	if err != nil {
		return Run{}, err
	}
	for _, run := range runs {
		if run.ID == id {
			return run, nil
		}
	}
	return Run{}, sql.ErrNoRows
}

func (s *Service) repositoryJob(ctx context.Context, job jobs.Job) (jobs.Job, error) {
	return s.repository.CreateJob(ctx, job)
}
