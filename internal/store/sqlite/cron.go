package sqlite

import (
	"context"
	"time"

	cronjob "github.com/GVALFER/WEBYCP/internal/cron"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite/dbgen"
)

func (s *Store) CreateCronJob(ctx context.Context, value cronjob.CronJob, job jobs.Job) (cronjob.CronJob, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return cronjob.CronJob{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	row, err := q.CreateCronJob(ctx, dbgen.CreateCronJobParams{
		ID: value.ID, AccountID: value.AccountID, NodeID: value.NodeID, Name: value.Name,
		Schedule: value.Schedule, Command: value.Command, Enabled: boolValue(value.Enabled),
		CreatedAt: timeValue(value.CreatedAt), UpdatedAt: timeValue(value.UpdatedAt),
	})
	if err != nil {
		return cronjob.CronJob{}, jobs.Job{}, err
	}
	created, err := createJob(ctx, q, job)
	if err != nil {
		return cronjob.CronJob{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return cronjob.CronJob{}, jobs.Job{}, err
	}
	return cronJobValue(row), created, nil
}

func (s *Store) UpdateCronJob(ctx context.Context, value cronjob.CronJob, job jobs.Job) (cronjob.CronJob, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return cronjob.CronJob{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	row, err := q.UpdateCronJob(ctx, dbgen.UpdateCronJobParams{
		Name: value.Name, Schedule: value.Schedule, Command: value.Command,
		Enabled: boolValue(value.Enabled), UpdatedAt: timeValue(value.UpdatedAt), ID: value.ID,
	})
	if err != nil {
		return cronjob.CronJob{}, jobs.Job{}, err
	}
	created, err := createJob(ctx, q, job)
	if err != nil {
		return cronjob.CronJob{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return cronjob.CronJob{}, jobs.Job{}, err
	}
	return cronJobValue(row), created, nil
}

func (s *Store) DeleteCronJob(ctx context.Context, id string, job jobs.Job) (jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	if err := q.DeleteCronJob(ctx, id); err != nil {
		return jobs.Job{}, err
	}
	created, err := createJob(ctx, q, job)
	if err != nil {
		return jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return jobs.Job{}, err
	}
	return created, nil
}

func (s *Store) CronJobs(ctx context.Context, userID string, admin bool) ([]cronjob.CronJob, error) {
	rows, err := s.queries.ListCronJobs(ctx, dbgen.ListCronJobsParams{Column1: admin, UserID: userID})
	if err != nil {
		return nil, err
	}
	return cronJobValues(rows), nil
}

func (s *Store) CronJob(ctx context.Context, id string) (cronjob.CronJob, error) {
	row, err := s.queries.GetCronJob(ctx, id)
	return cronJobValue(row), err
}

func (s *Store) AccountCronJobs(ctx context.Context, accountID string) ([]cronjob.CronJob, error) {
	rows, err := s.queries.ListAccountCronJobs(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return cronJobValues(rows), nil
}

func (s *Store) SetCronStatuses(ctx context.Context, accountID, enabledStatus, disabledStatus string) error {
	return s.queries.UpdateAccountCronStatuses(ctx, dbgen.UpdateAccountCronStatusesParams{
		Status: enabledStatus, Status_2: disabledStatus, UpdatedAt: timeValue(time.Now().UTC()), AccountID: accountID,
	})
}

func cronJobValues(rows []dbgen.CronJob) []cronjob.CronJob {
	result := make([]cronjob.CronJob, 0, len(rows))
	for _, row := range rows {
		result = append(result, cronJobValue(row))
	}
	return result
}

func cronJobValue(row dbgen.CronJob) cronjob.CronJob {
	return cronjob.CronJob{
		ID: row.ID, AccountID: row.AccountID, NodeID: row.NodeID, Name: row.Name,
		Schedule: row.Schedule, Command: row.Command, Enabled: row.Enabled != 0,
		Status: row.Status, CreatedAt: timeFrom(row.CreatedAt), UpdatedAt: timeFrom(row.UpdatedAt),
	}
}
