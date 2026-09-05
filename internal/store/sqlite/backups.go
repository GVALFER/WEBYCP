package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/GVALFER/WEBYCP/internal/backupfmt"
	"github.com/GVALFER/WEBYCP/internal/backups"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/services"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite/dbgen"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

func (s *Store) BackupPlans(ctx context.Context, userID string, admin bool) ([]backups.Plan, error) {
	rows, err := s.queries.ListBackupPlans(ctx, dbgen.ListBackupPlansParams{Column1: admin, UserID: userID})
	if err != nil {
		return nil, err
	}
	result := make([]backups.Plan, 0, len(rows))
	for _, row := range rows {
		result = append(result, backupPlanValue(row))
	}
	return result, nil
}

func (s *Store) BackupPlanPage(
	ctx context.Context, userID string, admin bool, query pagination.Query,
) (pagination.Result[backups.Plan], error) {
	total, err := s.queries.CountBackupPlans(ctx, dbgen.CountBackupPlansParams{
		IsAdmin: admin, UserID: userID,
	})
	if err != nil {
		return pagination.Result[backups.Plan]{}, err
	}
	query = pagination.Clamp(query, total)
	rows, err := s.queries.ListBackupPlansPage(ctx, dbgen.ListBackupPlansPageParams{
		IsAdmin: admin, UserID: userID,
		PageOffset: pagination.Offset(query), PageSize: int64(query.Size),
	})
	if err != nil {
		return pagination.Result[backups.Plan]{}, err
	}
	items := make([]backups.Plan, 0, len(rows))
	for _, row := range rows {
		items = append(items, backupPlanValue(row))
	}
	return pagination.Result[backups.Plan]{Items: items, Query: query, Total: total}, nil
}

func (s *Store) BackupPlan(ctx context.Context, id string) (backups.Plan, error) {
	row, err := s.queries.GetBackupPlan(ctx, id)
	return backupPlanValue(row), err
}

func (s *Store) CreateBackupPlan(ctx context.Context, value backups.Plan) (backups.Plan, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return backups.Plan{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	if err := requireCapacity(ctx, tx, value.AccountID, limitBackupPlans); err != nil {
		return backups.Plan{}, err
	}
	if err := requireRetention(ctx, tx, value.AccountID, value.RetentionCount, 0); err != nil {
		return backups.Plan{}, err
	}
	row, err := q.CreateBackupPlan(ctx, dbgen.CreateBackupPlanParams{
		ID: value.ID, AccountID: value.AccountID, NodeID: value.NodeID, Name: value.Name,
		Schedule: value.Schedule, RetentionCount: value.RetentionCount,
		StorageDriver: value.StorageDriver,
		IncludeFiles:  boolValue(value.IncludeFiles), IncludeDatabases: boolValue(value.IncludeDatabases),
		Enabled: boolValue(value.Enabled), NextRunAt: nullTime(value.NextRunAt),
		CreatedAt: timeValue(value.CreatedAt), UpdatedAt: timeValue(value.UpdatedAt),
	})
	if err != nil {
		return backups.Plan{}, err
	}
	if err := tx.Commit(); err != nil {
		return backups.Plan{}, err
	}
	return backupPlanValue(row), nil
}

func (s *Store) UpdateBackupPlan(ctx context.Context, value backups.Plan, currentRetention int64) (backups.Plan, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return backups.Plan{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	if err := requireRetention(ctx, tx, value.AccountID, value.RetentionCount, currentRetention); err != nil {
		return backups.Plan{}, err
	}
	row, err := q.UpdateBackupPlan(ctx, dbgen.UpdateBackupPlanParams{
		Name: value.Name, Schedule: value.Schedule, RetentionCount: value.RetentionCount,
		IncludeFiles: boolValue(value.IncludeFiles), IncludeDatabases: boolValue(value.IncludeDatabases),
		StorageDriver: value.StorageDriver,
		Enabled:       boolValue(value.Enabled), NextRunAt: nullTime(value.NextRunAt),
		UpdatedAt: timeValue(value.UpdatedAt), ID: value.ID,
	})
	if err != nil {
		return backups.Plan{}, err
	}
	if err := tx.Commit(); err != nil {
		return backups.Plan{}, err
	}
	return backupPlanValue(row), nil
}

func (s *Store) DeleteBackupPlan(ctx context.Context, id string) error {
	return s.queries.DeleteBackupPlan(ctx, id)
}

func (s *Store) DueBackupPlans(ctx context.Context, now time.Time) ([]backups.Plan, error) {
	rows, err := s.queries.ListDueBackupPlans(ctx, nullTime(&now))
	if err != nil {
		return nil, err
	}
	result := make([]backups.Plan, 0, len(rows))
	for _, row := range rows {
		result = append(result, backupPlanValue(row))
	}
	return result, nil
}

func (s *Store) BackupRunPending(ctx context.Context, planID string) (bool, error) {
	return s.queries.BackupRunPending(ctx, nullString(planID))
}

func (s *Store) QueueBackupRun(ctx context.Context, run backups.Run, job jobs.Job, next *time.Time) (backups.Run, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return backups.Run{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	row, err := q.CreateBackupRun(ctx, dbgen.CreateBackupRunParams{ID: run.ID, PlanID: nullString(run.PlanID), AccountID: run.AccountID, NodeID: run.NodeID, StorageDriver: run.StorageDriver, CreatedAt: timeValue(run.CreatedAt)})
	if err != nil {
		return backups.Run{}, jobs.Job{}, err
	}
	if err := q.MarkBackupPlanRun(ctx, dbgen.MarkBackupPlanRunParams{LastRunAt: nullTime(&run.CreatedAt), NextRunAt: nullTime(next), UpdatedAt: timeValue(run.CreatedAt), ID: run.PlanID}); err != nil {
		return backups.Run{}, jobs.Job{}, err
	}
	created, err := createJob(ctx, q, job)
	if err != nil {
		return backups.Run{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return backups.Run{}, jobs.Job{}, err
	}
	return backupRunValue(row), created, nil
}

func (s *Store) SetBackupRun(ctx context.Context, id, status, message string, started, finished *time.Time) error {
	return s.queries.UpdateBackupRun(ctx, dbgen.UpdateBackupRunParams{Status: status, Error: message, StartedAt: nullTime(started), FinishedAt: nullTime(finished), ID: id})
}

func (s *Store) CompleteBackup(ctx context.Context, run backups.Run, artifact backups.Artifact) (backups.Artifact, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return backups.Artifact{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	data, err := json.Marshal(artifact.Manifest)
	if err != nil {
		return backups.Artifact{}, err
	}
	row, err := q.CreateBackupArtifact(ctx, dbgen.CreateBackupArtifactParams{ID: artifact.ID, RunID: artifact.RunID, AccountID: artifact.AccountID, NodeID: artifact.NodeID, StorageDriver: artifact.StorageDriver, Path: artifact.Path, Checksum: artifact.Checksum, SizeBytes: artifact.Size, Manifest: string(data), CreatedAt: timeValue(artifact.CreatedAt)})
	if err != nil {
		return backups.Artifact{}, err
	}
	now := time.Now().UTC()
	if err := q.UpdateBackupRun(ctx, dbgen.UpdateBackupRunParams{Status: "succeeded", Error: "", FinishedAt: nullTime(&now), ID: run.ID}); err != nil {
		return backups.Artifact{}, err
	}
	if err := tx.Commit(); err != nil {
		return backups.Artifact{}, err
	}
	return backupArtifactValue(row)
}

func (s *Store) BackupRuns(ctx context.Context, userID string, admin bool) ([]backups.Run, error) {
	rows, err := s.queries.ListBackupRuns(ctx, dbgen.ListBackupRunsParams{Column1: admin, UserID: userID})
	if err != nil {
		return nil, err
	}
	result := make([]backups.Run, 0, len(rows))
	for _, row := range rows {
		result = append(result, backupRunValue(row))
	}
	return result, nil
}

func (s *Store) BackupRunPage(
	ctx context.Context, userID string, admin bool, query pagination.Query,
) (pagination.Result[backups.Run], error) {
	total, err := s.queries.CountBackupRuns(ctx, dbgen.CountBackupRunsParams{
		IsAdmin: admin, UserID: userID,
	})
	if err != nil {
		return pagination.Result[backups.Run]{}, err
	}
	query = pagination.Clamp(query, total)
	rows, err := s.queries.ListBackupRunsPage(ctx, dbgen.ListBackupRunsPageParams{
		IsAdmin: admin, UserID: userID,
		PageOffset: pagination.Offset(query), PageSize: int64(query.Size),
	})
	if err != nil {
		return pagination.Result[backups.Run]{}, err
	}
	items := make([]backups.Run, 0, len(rows))
	for _, row := range rows {
		items = append(items, backupRunValue(row))
	}
	return pagination.Result[backups.Run]{Items: items, Query: query, Total: total}, nil
}

func (s *Store) BackupArtifact(ctx context.Context, id string) (backups.Artifact, error) {
	row, err := s.queries.GetBackupArtifact(ctx, id)
	if err != nil {
		return backups.Artifact{}, err
	}
	return backupArtifactValue(row)
}

func (s *Store) BackupArtifacts(ctx context.Context, userID string, admin bool) ([]backups.Artifact, error) {
	rows, err := s.queries.ListBackupArtifacts(ctx, dbgen.ListBackupArtifactsParams{Column1: admin, UserID: userID})
	if err != nil {
		return nil, err
	}
	result := make([]backups.Artifact, 0, len(rows))
	for _, row := range rows {
		value, err := backupArtifactValue(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (s *Store) BackupArtifactPage(
	ctx context.Context, userID string, admin bool, query pagination.Query,
) (pagination.Result[backups.Artifact], error) {
	total, err := s.queries.CountBackupArtifacts(ctx, dbgen.CountBackupArtifactsParams{
		IsAdmin: admin, UserID: userID,
	})
	if err != nil {
		return pagination.Result[backups.Artifact]{}, err
	}
	query = pagination.Clamp(query, total)
	rows, err := s.queries.ListBackupArtifactsPage(ctx, dbgen.ListBackupArtifactsPageParams{
		IsAdmin: admin, UserID: userID,
		PageOffset: pagination.Offset(query), PageSize: int64(query.Size),
	})
	if err != nil {
		return pagination.Result[backups.Artifact]{}, err
	}
	items := make([]backups.Artifact, 0, len(rows))
	for _, row := range rows {
		value, valueErr := backupArtifactValue(row)
		if valueErr != nil {
			return pagination.Result[backups.Artifact]{}, valueErr
		}
		items = append(items, value)
	}
	return pagination.Result[backups.Artifact]{Items: items, Query: query, Total: total}, nil
}

func (s *Store) ExpiredBackupArtifacts(ctx context.Context, planID string, keep int64) ([]backups.Artifact, error) {
	rows, err := s.queries.ListExpiredBackupArtifacts(ctx, dbgen.ListExpiredBackupArtifactsParams{PlanID: nullString(planID), Offset: keep})
	if err != nil {
		return nil, err
	}
	result := make([]backups.Artifact, 0, len(rows))
	for _, row := range rows {
		value, err := backupArtifactValue(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (s *Store) DeleteBackupArtifact(ctx context.Context, id string) error {
	return s.queries.DeleteBackupArtifact(ctx, id)
}

func (s *Store) RestoreMetadata(ctx context.Context, value backupfmt.Metadata) error {
	if value.Version != backupfmt.Version || validate.ID("accountId", value.AccountID) != nil {
		return fmt.Errorf("restored metadata identity is invalid")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	account, err := q.GetAccount(ctx, value.AccountID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, website := range value.Websites {
		if validate.ID("websiteId", website.ID) != nil || !account.NodeID.Valid || website.NodeID != account.NodeID.String || website.Kind != "php" || website.WebDriver != services.Nginx || website.RuntimeDriver != services.PHPFPM || website.RuntimeVersion != services.PHP83 {
			return fmt.Errorf("restored website metadata is invalid")
		}
		if _, err := validate.ResourceName(website.Name); err != nil {
			return fmt.Errorf("restored website metadata is invalid")
		}
		base := filepath.Join("/home", account.SystemUser, "web")
		rel, relErr := filepath.Rel(base, filepath.Clean(website.DocumentRoot))
		parts := strings.Split(rel, string(filepath.Separator))
		if relErr != nil || filepath.IsAbs(rel) || len(parts) != 2 || parts[0] == "" || parts[0] == ".." || parts[1] != "public_html" {
			return fmt.Errorf("restored website document root is invalid")
		}
		current, getErr := q.GetWebsite(ctx, website.ID)
		if getErr == nil && current.AccountID != value.AccountID {
			return fmt.Errorf("restored website belongs to another account")
		}
		if getErr != nil && !errors.Is(getErr, sql.ErrNoRows) {
			return getErr
		}
		if errors.Is(getErr, sql.ErrNoRows) {
			if err := requireCapacity(ctx, tx, value.AccountID, limitWebsites); err != nil {
				return err
			}
		}
		enabled := boolValue(website.Enabled)
		if err := q.UpsertRestoredWebsite(ctx, dbgen.UpsertRestoredWebsiteParams{ID: website.ID, AccountID: value.AccountID, NodeID: website.NodeID, Name: website.Name, Kind: website.Kind, DocumentRoot: website.DocumentRoot, WebDriver: website.WebDriver, RuntimeDriver: website.RuntimeDriver, RuntimeVersion: website.RuntimeVersion, Column10: enabled, Enabled: enabled, CreatedAt: timeValue(now), UpdatedAt: timeValue(now)}); err != nil {
			return err
		}
		primary := 0
		for _, domain := range website.Domains {
			hostname, hostnameErr := validate.Domain(domain.Hostname)
			if validate.ID("websiteDomainId", domain.ID) != nil || hostnameErr != nil || hostname != domain.Hostname || (domain.Kind != "primary" && domain.Kind != "alias") {
				return fmt.Errorf("restored website domain metadata is invalid")
			}
			if domain.Kind == "primary" {
				primary++
			}
			current, getErr := q.GetWebsiteDomain(ctx, domain.ID)
			if getErr == nil && current.WebsiteID != website.ID {
				return fmt.Errorf("restored domain belongs to another website")
			}
			if getErr != nil && !errors.Is(getErr, sql.ErrNoRows) {
				return getErr
			}
			if errors.Is(getErr, sql.ErrNoRows) {
				kind := limitAliases
				if domain.Kind == "primary" {
					kind = limitDomains
				}
				if err := requireCapacity(ctx, tx, value.AccountID, kind); err != nil {
					return err
				}
			}
			domainEnabled := boolValue(domain.Enabled)
			if err := q.UpsertRestoredWebsiteDomain(ctx, dbgen.UpsertRestoredWebsiteDomainParams{ID: domain.ID, WebsiteID: website.ID, Hostname: domain.Hostname, Kind: domain.Kind, Column5: domainEnabled, Enabled: domainEnabled, CreatedAt: timeValue(now), UpdatedAt: timeValue(now)}); err != nil {
				return err
			}
		}
		if primary != 1 {
			return fmt.Errorf("restored website must have one primary domain")
		}
	}
	for _, database := range value.Databases {
		name, nameErr := validate.DatabaseName(database.Name)
		if validate.ID("databaseId", database.ID) != nil || !account.NodeID.Valid || database.NodeID != account.NodeID.String || nameErr != nil || name != database.Name || validate.DatabaseSystemName(database.SystemName) != nil || database.Driver != services.MySQL {
			return fmt.Errorf("restored database metadata is invalid")
		}
		current, getErr := q.GetDatabase(ctx, database.ID)
		if getErr == nil && current.AccountID != value.AccountID {
			return fmt.Errorf("restored database belongs to another account")
		}
		if getErr != nil && !errors.Is(getErr, sql.ErrNoRows) {
			return getErr
		}
		if errors.Is(getErr, sql.ErrNoRows) {
			if err := requireCapacity(ctx, tx, value.AccountID, limitDatabases); err != nil {
				return err
			}
		}
		if err := q.UpsertRestoredDatabase(ctx, dbgen.UpsertRestoredDatabaseParams{ID: database.ID, AccountID: value.AccountID, NodeID: database.NodeID, Name: database.Name, SystemName: database.SystemName, Driver: database.Driver, CreatedAt: timeValue(now), UpdatedAt: timeValue(now)}); err != nil {
			return err
		}
	}
	for _, task := range value.ScheduledTasks {
		if validate.ID("scheduledTaskId", task.ID) != nil || !account.NodeID.Valid || task.NodeID != account.NodeID.String || task.SchedulerDriver != services.Crontab || task.Kind != "command" {
			return fmt.Errorf("restored scheduled task metadata is invalid")
		}
		if _, err := validate.ResourceName(task.Name); err != nil {
			return err
		}
		if _, err := validate.CronSchedule(task.Schedule, false); err != nil {
			return err
		}
		if _, err := validate.CronCommand(task.Command); err != nil {
			return err
		}
		current, getErr := q.GetScheduledTask(ctx, task.ID)
		if getErr == nil && current.AccountID != value.AccountID {
			return fmt.Errorf("restored scheduled task belongs to another account")
		}
		if getErr != nil && !errors.Is(getErr, sql.ErrNoRows) {
			return getErr
		}
		if errors.Is(getErr, sql.ErrNoRows) {
			if err := requireCapacity(ctx, tx, value.AccountID, limitScheduledTasks); err != nil {
				return err
			}
		}
		enabled := boolValue(task.Enabled)
		if err := q.UpsertRestoredScheduledTask(ctx, dbgen.UpsertRestoredScheduledTaskParams{Kind: task.Kind, ID: task.ID, AccountID: value.AccountID, NodeID: task.NodeID, Name: task.Name, Schedule: task.Schedule, Command: task.Command, SchedulerDriver: task.SchedulerDriver, Enabled: enabled, Column10: enabled, CreatedAt: timeValue(now), UpdatedAt: timeValue(now)}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func backupPlanValue(row dbgen.BackupPlan) backups.Plan {
	return backups.Plan{ID: row.ID, AccountID: row.AccountID, NodeID: row.NodeID, Name: row.Name, Schedule: row.Schedule, RetentionCount: row.RetentionCount, StorageDriver: row.StorageDriver, IncludeFiles: row.IncludeFiles != 0, IncludeDatabases: row.IncludeDatabases != 0, Enabled: row.Enabled != 0, LastRunAt: timePtr(row.LastRunAt), NextRunAt: timePtr(row.NextRunAt), CreatedAt: timeFrom(row.CreatedAt), UpdatedAt: timeFrom(row.UpdatedAt)}
}

func backupRunValue(row dbgen.BackupRun) backups.Run {
	return backups.Run{ID: row.ID, PlanID: row.PlanID.String, AccountID: row.AccountID, NodeID: row.NodeID, StorageDriver: row.StorageDriver, Status: row.Status, Error: row.Error, CreatedAt: timeFrom(row.CreatedAt), StartedAt: timePtr(row.StartedAt), FinishedAt: timePtr(row.FinishedAt)}
}

func backupArtifactValue(row dbgen.BackupArtifact) (backups.Artifact, error) {
	var manifest backupfmt.Manifest
	if err := json.Unmarshal([]byte(row.Manifest), &manifest); err != nil {
		return backups.Artifact{}, fmt.Errorf("decode backup manifest: %w", err)
	}
	return backups.Artifact{ID: row.ID, RunID: row.RunID, AccountID: row.AccountID, NodeID: row.NodeID, StorageDriver: row.StorageDriver, Path: row.Path, Checksum: row.Checksum, Size: row.SizeBytes, Manifest: manifest, CreatedAt: timeFrom(row.CreatedAt)}, nil
}
