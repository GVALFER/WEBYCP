package sqlite

import (
	"context"

	"github.com/GVALFER/WEBYCP/internal/audit"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite/dbgen"
)

func (s *Store) Record(ctx context.Context, event audit.Event) error {
	return s.queries.CreateAuditEvent(ctx, dbgen.CreateAuditEventParams{
		ID:           event.ID,
		UserID:       nullString(event.UserID),
		Action:       event.Action,
		ResourceType: event.ResourceType,
		ResourceID:   nullString(event.ResourceID),
		JobID:        nullString(event.JobID),
		Result:       event.Result,
		Metadata:     event.Metadata,
		CreatedAt:    timeValue(event.CreatedAt),
	})
}

func (s *Store) AuditPage(ctx context.Context, query pagination.Query, jobID string) (pagination.Result[audit.Event], error) {
	total, err := s.queries.CountAuditEvents(ctx, jobID)
	if err != nil {
		return pagination.Result[audit.Event]{}, err
	}
	query = pagination.Clamp(query, total)
	rows, err := s.queries.ListAuditEventsPage(ctx, dbgen.ListAuditEventsPageParams{
		JobID: jobID, PageSize: int64(query.Size), PageOffset: pagination.Offset(query),
	})
	if err != nil {
		return pagination.Result[audit.Event]{}, err
	}
	items := make([]audit.Event, 0, len(rows))
	for _, row := range rows {
		items = append(items, audit.Event{
			ID: row.ID, UserID: row.UserID.String, Action: row.Action,
			ResourceType: row.ResourceType, ResourceID: row.ResourceID.String,
			JobID: row.JobID.String, Result: row.Result, CreatedAt: timeFrom(row.CreatedAt),
		})
	}
	return pagination.Result[audit.Event]{Items: items, Query: query, Total: total}, nil
}
