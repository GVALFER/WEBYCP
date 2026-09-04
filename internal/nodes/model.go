package nodes

import (
	"context"
	"time"

	"github.com/GVALFER/WEBYCP/internal/services"
)

type Node struct {
	ID             string
	Name           string
	Kind           string
	Endpoint       string
	Status         string
	LastSeenAt     *time.Time
	Capabilities   *services.Capabilities
	CapabilitiesAt *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Repository interface {
	EnsureLocal(context.Context, string, string) (Node, error)
	Node(context.Context, string) (Node, error)
	Nodes(context.Context) ([]Node, error)
	UpdateProbe(context.Context, string, string, *time.Time, *services.Capabilities) error
}
