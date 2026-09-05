package sqlite

import (
	"context"
	"time"

	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite/dbgen"
	"github.com/GVALFER/WEBYCP/internal/tasks"
)

func (s *Store) CreateScheduledTask(ctx context.Context, value tasks.ScheduledTask, job jobs.Job) (tasks.ScheduledTask, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return tasks.ScheduledTask{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	if err := requireCapacity(ctx, tx, value.AccountID, limitScheduledTasks); err != nil {
		return tasks.ScheduledTask{}, jobs.Job{}, err
	}
	row, err := q.CreateScheduledTask(ctx, dbgen.CreateScheduledTaskParams{
		ID: value.ID, AccountID: value.AccountID, NodeID: value.NodeID, Name: value.Name,
		Schedule: value.Schedule, Command: value.Command, SchedulerDriver: value.SchedulerDriver,
		Kind:      string(value.Kind),
		Enabled:   boolValue(value.Enabled),
		CreatedAt: timeValue(value.CreatedAt), UpdatedAt: timeValue(value.UpdatedAt),
	})
	if err != nil {
		return tasks.ScheduledTask{}, jobs.Job{}, err
	}
	created, err := createJob(ctx, q, job)
	if err != nil {
		return tasks.ScheduledTask{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return tasks.ScheduledTask{}, jobs.Job{}, err
	}
	return scheduledTaskValue(row), created, nil
}

func (s *Store) UpdateScheduledTask(ctx context.Context, value tasks.ScheduledTask, job jobs.Job) (tasks.ScheduledTask, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return tasks.ScheduledTask{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	row, err := q.UpdateScheduledTask(ctx, dbgen.UpdateScheduledTaskParams{
		Name: value.Name, Schedule: value.Schedule, Command: value.Command,
		SchedulerDriver: value.SchedulerDriver, Enabled: boolValue(value.Enabled),
		Kind:      string(value.Kind),
		UpdatedAt: timeValue(value.UpdatedAt), ID: value.ID,
	})
	if err != nil {
		return tasks.ScheduledTask{}, jobs.Job{}, err
	}
	created, err := createJob(ctx, q, job)
	if err != nil {
		return tasks.ScheduledTask{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return tasks.ScheduledTask{}, jobs.Job{}, err
	}
	return scheduledTaskValue(row), created, nil
}

func (s *Store) DeleteScheduledTask(ctx context.Context, id string, job jobs.Job) (jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	if err := q.DeleteScheduledTask(ctx, id); err != nil {
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

func (s *Store) ScheduledTasks(ctx context.Context, userID string, admin bool) ([]tasks.ScheduledTask, error) {
	rows, err := s.queries.ListScheduledTasks(ctx, dbgen.ListScheduledTasksParams{Column1: admin, UserID: userID})
	if err != nil {
		return nil, err
	}
	return scheduledTaskValues(rows), nil
}

func (s *Store) ScheduledTaskPage(
	ctx context.Context, userID string, admin bool, query pagination.Query,
) (pagination.Result[tasks.ScheduledTask], error) {
	total, err := s.queries.CountScheduledTasks(ctx, dbgen.CountScheduledTasksParams{
		IsAdmin: admin, UserID: userID,
	})
	if err != nil {
		return pagination.Result[tasks.ScheduledTask]{}, err
	}
	query = pagination.Clamp(query, total)
	rows, err := s.queries.ListScheduledTasksPage(ctx, dbgen.ListScheduledTasksPageParams{
		IsAdmin: admin, UserID: userID,
		PageOffset: pagination.Offset(query), PageSize: int64(query.Size),
	})
	if err != nil {
		return pagination.Result[tasks.ScheduledTask]{}, err
	}
	return pagination.Result[tasks.ScheduledTask]{
		Items: scheduledTaskValues(rows), Query: query, Total: total,
	}, nil
}

func (s *Store) ScheduledTask(ctx context.Context, id string) (tasks.ScheduledTask, error) {
	row, err := s.queries.GetScheduledTask(ctx, id)
	return scheduledTaskValue(row), err
}

func (s *Store) AccountScheduledTasks(ctx context.Context, accountID string) ([]tasks.ScheduledTask, error) {
	rows, err := s.queries.ListAccountScheduledTasks(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return scheduledTaskValues(rows), nil
}

func (s *Store) SetTaskStatuses(ctx context.Context, accountID, enabledStatus, disabledStatus string) error {
	return s.queries.UpdateAccountTaskStatuses(ctx, dbgen.UpdateAccountTaskStatusesParams{
		Status: enabledStatus, Status_2: disabledStatus, UpdatedAt: timeValue(time.Now().UTC()), AccountID: accountID,
	})
}

func scheduledTaskValues(rows []dbgen.ScheduledTask) []tasks.ScheduledTask {
	result := make([]tasks.ScheduledTask, 0, len(rows))
	for _, row := range rows {
		result = append(result, scheduledTaskValue(row))
	}
	return result
}

func scheduledTaskValue(row dbgen.ScheduledTask) tasks.ScheduledTask {
	return tasks.ScheduledTask{
		Kind: tasks.Kind(row.Kind),
		ID:   row.ID, AccountID: row.AccountID, NodeID: row.NodeID, Name: row.Name,
		Schedule: row.Schedule, Command: row.Command, SchedulerDriver: row.SchedulerDriver,
		Enabled: row.Enabled != 0,
		Status:  row.Status, CreatedAt: timeFrom(row.CreatedAt), UpdatedAt: timeFrom(row.UpdatedAt),
	}
}
