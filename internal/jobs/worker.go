package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/GVALFER/WEBYCP/internal/audit"
	"github.com/GVALFER/WEBYCP/internal/idgen"
)

type Handler func(context.Context, Job) error

type Worker struct {
	repository Repository
	audit      audit.Recorder
	handlers   map[string]Handler
	logger     *slog.Logger
	wake       chan struct{}
}

func NewWorker(repository Repository, recorder audit.Recorder, logger *slog.Logger) *Worker {
	return &Worker{
		repository: repository,
		audit:      recorder,
		handlers:   make(map[string]Handler),
		logger:     logger,
		wake:       make(chan struct{}, 1),
	}
}

func (w *Worker) Handle(kind string, handler Handler) {
	w.handlers[kind] = handler
}

func (w *Worker) Notify() {
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

func (w *Worker) Run(ctx context.Context) error {
	if err := w.repository.RecoverJobs(ctx, time.Now().UTC()); err != nil {
		return fmt.Errorf("recover jobs: %w", err)
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		if retry := w.drain(ctx); retry {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
			}
			continue
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		case <-w.wake:
		}
	}
}

func (w *Worker) drain(ctx context.Context) bool {
	for {
		job, err := w.repository.ClaimJob(ctx, time.Now().UTC())
		if errors.Is(err, ErrNone) {
			return false
		}
		if err != nil {
			w.logger.Error("failed to claim job", "error", err)
			return true
		}

		if retry := w.run(ctx, job); retry {
			return true
		}
	}
}

func (w *Worker) run(ctx context.Context, job Job) bool {
	stepID, err := idgen.ID()
	if err != nil {
		w.fail(ctx, job, "generate job step id")
		return false
	}
	step, err := w.repository.CreateStep(ctx, Step{
		ID:        stepID,
		JobID:     job.ID,
		Name:      job.Kind,
		Status:    "running",
		StartedAt: time.Now().UTC(),
	})
	if err != nil {
		w.fail(ctx, job, "create job step")
		return false
	}

	handler, ok := w.handlers[job.Kind]
	if !ok {
		err = fmt.Errorf("unsupported job kind: %s", job.Kind)
	} else {
		err = handler(ctx, job)
	}

	now := time.Now().UTC()
	if err == nil {
		_ = w.repository.FinishStep(ctx, step.ID, "succeeded", "", now)
		if completeErr := w.repository.CompleteJob(ctx, job.ID, now); completeErr != nil {
			w.logger.Error("failed to complete job", "error", completeErr, "jobId", job.ID)
		} else {
			w.record(ctx, job, "success")
		}
		return false
	}

	message := safeError(err)
	_ = w.repository.FinishStep(ctx, step.ID, "failed", message, now)
	if job.Attempts < job.MaxAttempts {
		if retryErr := w.repository.RetryJob(ctx, job.ID, message); retryErr != nil {
			w.logger.Error("failed to retry job", "error", retryErr, "jobId", job.ID)
		}
		return true
	}

	w.fail(ctx, job, message)
	return false
}

func (w *Worker) fail(ctx context.Context, job Job, message string) {
	if err := w.repository.FailJob(ctx, job.ID, message, time.Now().UTC()); err != nil {
		w.logger.Error("failed to mark job as failed", "error", err, "jobId", job.ID)
		return
	}
	w.record(ctx, job, "failure")
}

func (w *Worker) record(ctx context.Context, job Job, result string) {
	if w.audit == nil {
		return
	}
	id, err := idgen.ID()
	if err != nil {
		w.logger.Error("failed to generate audit id", "error", err, "jobId", job.ID)
		return
	}
	if err := w.audit.Record(ctx, audit.Event{
		ID: id, UserID: job.UserID, Action: "job.execute", ResourceType: "job",
		ResourceID: job.ID, Result: result, Metadata: "{}", CreatedAt: time.Now().UTC(),
	}); err != nil {
		w.logger.Error("failed to record job audit event", "error", err, "jobId", job.ID)
	}
}

func safeError(err error) string {
	message := err.Error()
	if len(message) > 500 {
		return message[:500]
	}

	return message
}
