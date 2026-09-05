package sqlite

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/GVALFER/WEBYCP/internal/audit"
	"github.com/GVALFER/WEBYCP/internal/pagination"
)

func TestAuditPaginationAndJobFilter(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	stamp := time.Now().UTC().Truncate(time.Millisecond)
	for i := 1; i <= 13; i++ {
		event := audit.Event{ID: fmt.Sprintf("%032x", i), Action: "task.create", ResourceType: "scheduled_task", Result: "success", Metadata: `{"command":"must-not-be-listed"}`, CreatedAt: stamp}
		if i >= 12 {
			event.JobID = "0123456789abcdef0123456789abcdef"
		}
		if i == 13 {
			event.Action = "job.execute"
		}
		if err := store.Record(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	page, err := store.AuditPage(ctx, pagination.Query{Page: 2, Size: 10}, "")
	if err != nil || page.Total != 13 || len(page.Items) != 3 || page.Items[0].ID != fmt.Sprintf("%032x", 3) {
		t.Fatalf("page = %+v, error = %v", page, err)
	}
	filtered, err := store.AuditPage(ctx, pagination.Query{Page: 9, Size: 10}, "0123456789abcdef0123456789abcdef")
	if err != nil || filtered.Total != 2 || filtered.Query.Page != 1 {
		t.Fatalf("filtered = %+v, error = %v", filtered, err)
	}
	if filtered.Items[0].Action != "job.execute" || filtered.Items[1].Action != "task.create" {
		t.Fatalf("events = %+v", filtered.Items)
	}
	for _, event := range filtered.Items {
		if event.Metadata != "" {
			t.Fatal("audit listing exposed secret metadata")
		}
	}
	empty, err := store.AuditPage(ctx, pagination.Query{Page: 1, Size: 10}, "abcdef0123456789abcdef0123456789")
	if err != nil || empty.Total != 0 || len(empty.Items) != 0 {
		t.Fatalf("empty = %+v, error = %v", empty, err)
	}
}
