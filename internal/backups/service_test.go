package backups

import (
	"testing"
	"time"
)

func TestPreparePlanRejectsUnsupportedDriver(t *testing.T) {
	value := Plan{
		Name: "Daily", RetentionCount: 7, StorageDriver: "s3", IncludeFiles: true,
	}
	if err := preparePlan(&value); err == nil {
		t.Fatal("unsupported backup driver was accepted")
	}
}

func TestNextRun(t *testing.T) {
	now := time.Date(2026, 9, 5, 3, 0, 0, 0, time.UTC)
	next := nextRun("0 3 * * *", true, now)
	want := now.Add(24 * time.Hour)
	if next == nil || !next.Equal(want) || next.Location() != time.UTC {
		t.Fatalf("nextRun() = %v, want %v", next, want)
	}
	if nextRun("", true, now) != nil || nextRun("0 3 * * *", false, now) != nil {
		t.Fatal("manual and disabled plans must not be scheduled")
	}
}
