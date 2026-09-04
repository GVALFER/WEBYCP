package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/GVALFER/WEBYCP/internal/idgen"
	"github.com/GVALFER/WEBYCP/internal/pagination"
)

type Service struct {
	repository Repository
	notify     func()
}

func NewService(repository Repository, notify func()) *Service {
	return &Service{repository: repository, notify: notify}
}

func (s *Service) QueueProbe(ctx context.Context, nodeID, userID string) (Job, error) {
	id, err := idgen.ID()
	if err != nil {
		return Job{}, err
	}

	job, err := s.repository.CreateJob(ctx, Job{
		ID:          id,
		NodeID:      nodeID,
		UserID:      userID,
		Kind:        KindNodeProbe,
		Status:      "queued",
		Payload:     "{}",
		MaxAttempts: 2,
		CreatedAt:   time.Now().UTC(),
	})
	if err != nil {
		return Job{}, fmt.Errorf("queue node probe: %w", err)
	}

	s.notify()
	return job, nil
}

func (s *Service) Job(ctx context.Context, id string) (Job, []Step, error) {
	job, err := s.repository.Job(ctx, id)
	if err != nil {
		return Job{}, nil, err
	}
	steps, err := s.repository.Steps(ctx, id)
	if err != nil {
		return Job{}, nil, err
	}

	return job, steps, nil
}

func (s *Service) Jobs(ctx context.Context) ([]Job, error) {
	return s.repository.Jobs(ctx, 50)
}

func (s *Service) JobPage(
	ctx context.Context, query pagination.Query,
) (pagination.Result[Job], error) {
	return s.repository.JobPage(ctx, query)
}
