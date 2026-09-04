package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/GVALFER/WEBYCP/internal/backupfmt"
	"github.com/GVALFER/WEBYCP/internal/backups"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/pagination"
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
	row, err := s.queries.CreateBackupPlan(ctx, dbgen.CreateBackupPlanParams{
		ID: value.ID, AccountID: value.AccountID, NodeID: value.NodeID, Name: value.Name,
		Schedule: value.Schedule, RetentionCount: value.RetentionCount,
		IncludeFiles: boolValue(value.IncludeFiles), IncludeDatabases: boolValue(value.IncludeDatabases),
		Enabled: boolValue(value.Enabled), NextRunAt: nullTime(value.NextRunAt),
		CreatedAt: timeValue(value.CreatedAt), UpdatedAt: timeValue(value.UpdatedAt),
	})
	return backupPlanValue(row), err
}

func (s *Store) UpdateBackupPlan(ctx context.Context, value backups.Plan) (backups.Plan, error) {
	row, err := s.queries.UpdateBackupPlan(ctx, dbgen.UpdateBackupPlanParams{
		Name: value.Name, Schedule: value.Schedule, RetentionCount: value.RetentionCount,
		IncludeFiles: boolValue(value.IncludeFiles), IncludeDatabases: boolValue(value.IncludeDatabases),
		Enabled: boolValue(value.Enabled), NextRunAt: nullTime(value.NextRunAt),
		UpdatedAt: timeValue(value.UpdatedAt), ID: value.ID,
	})
	return backupPlanValue(row), err
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
	row, err := q.CreateBackupRun(ctx, dbgen.CreateBackupRunParams{ID: run.ID, PlanID: nullString(run.PlanID), AccountID: run.AccountID, NodeID: run.NodeID, CreatedAt: timeValue(run.CreatedAt)})
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
	row, err := q.CreateBackupArtifact(ctx, dbgen.CreateBackupArtifactParams{ID: artifact.ID, RunID: artifact.RunID, AccountID: artifact.AccountID, NodeID: artifact.NodeID, Path: artifact.Path, Checksum: artifact.Checksum, SizeBytes: artifact.Size, Manifest: string(data), CreatedAt: timeValue(artifact.CreatedAt)})
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
	for _, domain := range value.Domains {
		name, nameErr := validate.Domain(domain.Name)
		if validate.ID("domainId", domain.ID) != nil || !account.NodeID.Valid || domain.NodeID != account.NodeID.String || nameErr != nil || name != domain.Name || domain.PHPVersion != "8.3" {
			return fmt.Errorf("restored domain metadata is invalid")
		}
		current, getErr := q.GetDomain(ctx, domain.ID)
		if getErr == nil && current.AccountID != value.AccountID {
			return fmt.Errorf("restored domain belongs to another account")
		}
		if getErr != nil && !errors.Is(getErr, sql.ErrNoRows) {
			return getErr
		}
		if err := q.UpsertRestoredDomain(ctx, dbgen.UpsertRestoredDomainParams{ID: domain.ID, AccountID: value.AccountID, NodeID: domain.NodeID, Name: domain.Name, PhpVersion: domain.PHPVersion, Enabled: boolValue(domain.Enabled), CreatedAt: timeValue(now), UpdatedAt: timeValue(now)}); err != nil {
			return err
		}
		for _, alias := range domain.Aliases {
			name, nameErr := validate.Domain(alias.Name)
			if validate.ID("aliasId", alias.ID) != nil || nameErr != nil || name != alias.Name {
				return fmt.Errorf("restored alias metadata is invalid")
			}
			current, getErr := q.GetDomainAlias(ctx, alias.ID)
			if getErr == nil && current.DomainID != domain.ID {
				return fmt.Errorf("restored alias belongs to another domain")
			}
			if getErr != nil && !errors.Is(getErr, sql.ErrNoRows) {
				return getErr
			}
			enabled := boolValue(alias.Enabled)
			if err := q.UpsertRestoredAlias(ctx, dbgen.UpsertRestoredAliasParams{ID: alias.ID, DomainID: domain.ID, Name: alias.Name, Column4: enabled, Enabled: enabled, CreatedAt: timeValue(now), UpdatedAt: timeValue(now)}); err != nil {
				return err
			}
		}
	}
	for _, database := range value.Databases {
		name, nameErr := validate.DatabaseName(database.Name)
		if validate.ID("databaseId", database.ID) != nil || !account.NodeID.Valid || database.NodeID != account.NodeID.String || nameErr != nil || name != database.Name || validate.DatabaseSystemName(database.SystemName) != nil {
			return fmt.Errorf("restored database metadata is invalid")
		}
		current, getErr := q.GetDatabase(ctx, database.ID)
		if getErr == nil && current.AccountID != value.AccountID {
			return fmt.Errorf("restored database belongs to another account")
		}
		if getErr != nil && !errors.Is(getErr, sql.ErrNoRows) {
			return getErr
		}
		if err := q.UpsertRestoredDatabase(ctx, dbgen.UpsertRestoredDatabaseParams{ID: database.ID, AccountID: value.AccountID, NodeID: database.NodeID, Name: database.Name, SystemName: database.SystemName, CreatedAt: timeValue(now), UpdatedAt: timeValue(now)}); err != nil {
			return err
		}
	}
	for _, cron := range value.CronJobs {
		if validate.ID("cronJobId", cron.ID) != nil || !account.NodeID.Valid || cron.NodeID != account.NodeID.String {
			return fmt.Errorf("restored cron metadata is invalid")
		}
		if _, err := validate.ResourceName(cron.Name); err != nil {
			return err
		}
		if _, err := validate.CronSchedule(cron.Schedule, false); err != nil {
			return err
		}
		if _, err := validate.CronCommand(cron.Command); err != nil {
			return err
		}
		current, getErr := q.GetCronJob(ctx, cron.ID)
		if getErr == nil && current.AccountID != value.AccountID {
			return fmt.Errorf("restored cron job belongs to another account")
		}
		if getErr != nil && !errors.Is(getErr, sql.ErrNoRows) {
			return getErr
		}
		enabled := boolValue(cron.Enabled)
		if err := q.UpsertRestoredCronJob(ctx, dbgen.UpsertRestoredCronJobParams{ID: cron.ID, AccountID: value.AccountID, NodeID: cron.NodeID, Name: cron.Name, Schedule: cron.Schedule, Command: cron.Command, Enabled: enabled, Column8: enabled, CreatedAt: timeValue(now), UpdatedAt: timeValue(now)}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func backupPlanValue(row dbgen.BackupPlan) backups.Plan {
	return backups.Plan{ID: row.ID, AccountID: row.AccountID, NodeID: row.NodeID, Name: row.Name, Schedule: row.Schedule, RetentionCount: row.RetentionCount, IncludeFiles: row.IncludeFiles != 0, IncludeDatabases: row.IncludeDatabases != 0, Enabled: row.Enabled != 0, LastRunAt: timePtr(row.LastRunAt), NextRunAt: timePtr(row.NextRunAt), CreatedAt: timeFrom(row.CreatedAt), UpdatedAt: timeFrom(row.UpdatedAt)}
}

func backupRunValue(row dbgen.BackupRun) backups.Run {
	return backups.Run{ID: row.ID, PlanID: row.PlanID.String, AccountID: row.AccountID, NodeID: row.NodeID, Status: row.Status, Error: row.Error, CreatedAt: timeFrom(row.CreatedAt), StartedAt: timePtr(row.StartedAt), FinishedAt: timePtr(row.FinishedAt)}
}

func backupArtifactValue(row dbgen.BackupArtifact) (backups.Artifact, error) {
	var manifest backupfmt.Manifest
	if err := json.Unmarshal([]byte(row.Manifest), &manifest); err != nil {
		return backups.Artifact{}, fmt.Errorf("decode backup manifest: %w", err)
	}
	return backups.Artifact{ID: row.ID, RunID: row.RunID, AccountID: row.AccountID, NodeID: row.NodeID, Path: row.Path, Checksum: row.Checksum, Size: row.SizeBytes, Manifest: manifest, CreatedAt: timeFrom(row.CreatedAt)}, nil
}
