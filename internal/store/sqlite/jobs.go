package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite/dbgen"
)

func (s *Store) CreateJob(ctx context.Context, job jobs.Job) (jobs.Job, error) {
	row, err := s.queries.CreateJob(ctx, dbgen.CreateJobParams{
		ID:          job.ID,
		NodeID:      nullString(job.NodeID),
		UserID:      nullString(job.UserID),
		Kind:        job.Kind,
		Payload:     job.Payload,
		MaxAttempts: job.MaxAttempts,
		CreatedAt:   timeValue(job.CreatedAt),
	})
	if err != nil {
		return jobs.Job{}, err
	}
	return jobValue(row), nil
}

func (s *Store) Job(ctx context.Context, id string) (jobs.Job, error) {
	row, err := s.queries.GetJob(ctx, id)
	if err != nil {
		return jobs.Job{}, err
	}
	return jobValue(row), nil
}

func (s *Store) Jobs(ctx context.Context, limit int64) ([]jobs.Job, error) {
	rows, err := s.queries.ListJobs(ctx, limit)
	if err != nil {
		return nil, err
	}
	result := make([]jobs.Job, 0, len(rows))
	for _, row := range rows {
		result = append(result, jobValue(row))
	}
	return result, nil
}

func (s *Store) ClaimJob(ctx context.Context, now time.Time) (jobs.Job, error) {
	row, err := s.queries.ClaimJob(ctx, nullTime(&now))
	if errors.Is(err, sql.ErrNoRows) {
		return jobs.Job{}, jobs.ErrNone
	}
	if err != nil {
		return jobs.Job{}, err
	}
	return jobValue(row), nil
}

func (s *Store) CompleteJob(ctx context.Context, id string, finishedAt time.Time) error {
	return s.queries.CompleteJob(ctx, dbgen.CompleteJobParams{
		FinishedAt: nullTime(&finishedAt),
		ID:         id,
	})
}

func (s *Store) RetryJob(ctx context.Context, id, message string) error {
	return s.queries.RetryJob(ctx, dbgen.RetryJobParams{
		Error: nullString(message),
		ID:    id,
	})
}

func (s *Store) FailJob(ctx context.Context, id, message string, finishedAt time.Time) error {
	return s.queries.FailJob(ctx, dbgen.FailJobParams{
		FinishedAt: nullTime(&finishedAt),
		Error:      nullString(message),
		ID:         id,
	})
}

func (s *Store) RecoverJobs(ctx context.Context, _ time.Time) error {
	_, err := s.queries.RecoverJobs(ctx)
	return err
}

func (s *Store) CreateStep(ctx context.Context, step jobs.Step) (jobs.Step, error) {
	row, err := s.queries.CreateJobStep(ctx, dbgen.CreateJobStepParams{
		ID:        step.ID,
		JobID:     step.JobID,
		Name:      step.Name,
		StartedAt: timeValue(step.StartedAt),
	})
	if err != nil {
		return jobs.Step{}, err
	}
	return stepValue(row), nil
}

func (s *Store) FinishStep(ctx context.Context, id, status, message string, finishedAt time.Time) error {
	return s.queries.FinishJobStep(ctx, dbgen.FinishJobStepParams{
		Status:     status,
		Message:    message,
		FinishedAt: nullTime(&finishedAt),
		ID:         id,
	})
}

func (s *Store) Steps(ctx context.Context, jobID string) ([]jobs.Step, error) {
	rows, err := s.queries.ListJobSteps(ctx, jobID)
	if err != nil {
		return nil, err
	}
	result := make([]jobs.Step, 0, len(rows))
	for _, row := range rows {
		result = append(result, stepValue(row))
	}
	return result, nil
}

func jobValue(row dbgen.Job) jobs.Job {
	return jobs.Job{
		ID:          row.ID,
		NodeID:      row.NodeID.String,
		UserID:      row.UserID.String,
		Kind:        row.Kind,
		Status:      row.Status,
		Payload:     row.Payload,
		Attempts:    row.Attempts,
		MaxAttempts: row.MaxAttempts,
		Error:       row.Error.String,
		CreatedAt:   timeFrom(row.CreatedAt),
		StartedAt:   timePtr(row.StartedAt),
		FinishedAt:  timePtr(row.FinishedAt),
	}
}

func stepValue(row dbgen.JobStep) jobs.Step {
	return jobs.Step{
		ID:         row.ID,
		JobID:      row.JobID,
		Name:       row.Name,
		Status:     row.Status,
		Message:    row.Message,
		StartedAt:  timeFrom(row.StartedAt),
		FinishedAt: timePtr(row.FinishedAt),
	}
}
