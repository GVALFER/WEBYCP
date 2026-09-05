package audit

import (
	"context"
	"time"

	"github.com/GVALFER/WEBYCP/internal/pagination"
)

type Event struct {
	ID           string
	UserID       string
	Action       string
	ResourceType string
	ResourceID   string
	JobID        string
	Result       string
	Metadata     string
	CreatedAt    time.Time
}

type Recorder interface {
	Record(context.Context, Event) error
}

type Repository interface {
	Recorder
	AuditPage(context.Context, pagination.Query, string) (pagination.Result[Event], error)
}
