package nodes

import (
	"context"
	"fmt"
	"time"

	"github.com/GVALFER/WEBYCP/internal/services"
)

type Prober interface {
	Probe(context.Context, string) (services.Capabilities, error)
}

type Service struct {
	repository Repository
	prober     Prober
}

func NewService(repository Repository, prober Prober) *Service {
	return &Service{repository: repository, prober: prober}
}

func (s *Service) Nodes(ctx context.Context) ([]Node, error) {
	return s.repository.Nodes(ctx)
}

func (s *Service) Node(ctx context.Context, id string) (Node, error) {
	return s.repository.Node(ctx, id)
}

func (s *Service) Probe(ctx context.Context, id string) error {
	node, err := s.repository.Node(ctx, id)
	if err != nil {
		return fmt.Errorf("get node: %w", err)
	}
	capabilities, err := s.prober.Probe(ctx, node.Endpoint)
	if err != nil {
		if updateErr := s.repository.UpdateProbe(ctx, node.ID, "offline", nil, nil); updateErr != nil {
			return fmt.Errorf("probe agent: %v; mark node offline: %w", err, updateErr)
		}
		return err
	}

	now := time.Now().UTC()
	if err := s.repository.UpdateProbe(ctx, node.ID, "online", &now, &capabilities); err != nil {
		return fmt.Errorf("mark node online: %w", err)
	}
	return nil
}
