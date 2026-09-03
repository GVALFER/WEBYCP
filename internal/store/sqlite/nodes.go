package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/GVALFER/WEBYCP/internal/idgen"
	"github.com/GVALFER/WEBYCP/internal/nodes"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite/dbgen"
)

func (s *Store) EnsureLocal(ctx context.Context, name, endpoint string) (nodes.Node, error) {
	_, err := s.queries.GetLocalNode(ctx)
	if err == nil {
		updated, updateErr := s.queries.UpdateLocalNode(ctx, dbgen.UpdateLocalNodeParams{
			Name:      name,
			Endpoint:  endpoint,
			UpdatedAt: timeValue(time.Now().UTC()),
		})
		if updateErr != nil {
			return nodes.Node{}, fmt.Errorf("update local node: %w", updateErr)
		}
		return nodeValue(updated), nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nodes.Node{}, fmt.Errorf("get local node: %w", err)
	}

	id, err := idgen.ID()
	if err != nil {
		return nodes.Node{}, err
	}
	now := time.Now().UTC()
	created, err := s.queries.CreateNode(ctx, dbgen.CreateNodeParams{
		ID:        id,
		Name:      name,
		Kind:      "local",
		Endpoint:  endpoint,
		Status:    "unknown",
		CreatedAt: timeValue(now),
		UpdatedAt: timeValue(now),
	})
	if err != nil {
		return nodes.Node{}, fmt.Errorf("create local node: %w", err)
	}
	return nodeValue(created), nil
}

func (s *Store) Node(ctx context.Context, id string) (nodes.Node, error) {
	node, err := s.queries.GetNode(ctx, id)
	if err != nil {
		return nodes.Node{}, err
	}
	return nodeValue(node), nil
}

func (s *Store) Nodes(ctx context.Context) ([]nodes.Node, error) {
	rows, err := s.queries.ListNodes(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]nodes.Node, 0, len(rows))
	for _, row := range rows {
		result = append(result, nodeValue(row))
	}
	return result, nil
}

func (s *Store) UpdateProbe(ctx context.Context, id, status string, seenAt *time.Time) error {
	return s.queries.UpdateNodeProbe(ctx, dbgen.UpdateNodeProbeParams{
		Status:     status,
		LastSeenAt: nullTime(seenAt),
		UpdatedAt:  timeValue(time.Now().UTC()),
		ID:         id,
	})
}

func nodeValue(node dbgen.Node) nodes.Node {
	return nodes.Node{
		ID:         node.ID,
		Name:       node.Name,
		Kind:       node.Kind,
		Endpoint:   node.Endpoint,
		Status:     node.Status,
		LastSeenAt: timePtr(node.LastSeenAt),
		CreatedAt:  timeFrom(node.CreatedAt),
		UpdatedAt:  timeFrom(node.UpdatedAt),
	}
}
