package sqlite

import (
	"context"

	"github.com/GVALFER/WEBYCP/internal/audit"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite/dbgen"
)

func (s *Store) Record(ctx context.Context, event audit.Event) error {
	return s.queries.CreateAuditEvent(ctx, dbgen.CreateAuditEventParams{
		ID:           event.ID,
		UserID:       nullString(event.UserID),
		Action:       event.Action,
		ResourceType: event.ResourceType,
		ResourceID:   nullString(event.ResourceID),
		Result:       event.Result,
		Metadata:     event.Metadata,
		CreatedAt:    timeValue(event.CreatedAt),
	})
}
