package audit

import (
	"context"
	"time"
)

type Event struct {
	ID           string
	UserID       string
	Action       string
	ResourceType string
	ResourceID   string
	Result       string
	Metadata     string
	CreatedAt    time.Time
}

type Recorder interface {
	Record(context.Context, Event) error
}
