package backups

import "testing"

func TestPreparePlanRejectsUnsupportedDriver(t *testing.T) {
	value := Plan{
		Name: "Daily", RetentionCount: 7, StorageDriver: "s3", IncludeFiles: true,
	}
	if err := preparePlan(&value); err == nil {
		t.Fatal("unsupported backup driver was accepted")
	}
}
