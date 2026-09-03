package nodes

import (
	"context"
	"time"
)

type Node struct {
	ID         string
	Name       string
	Kind       string
	Endpoint   string
	Status     string
	LastSeenAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Repository interface {
	EnsureLocal(context.Context, string, string) (Node, error)
	Node(context.Context, string) (Node, error)
	Nodes(context.Context) ([]Node, error)
	UpdateProbe(context.Context, string, string, *time.Time) error
}
