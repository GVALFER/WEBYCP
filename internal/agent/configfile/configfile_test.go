package configfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotRestoresExistingAndNewFiles(t *testing.T) {
	root := t.TempDir()
	existing := filepath.Join(root, "existing.conf")
	if err := os.WriteFile(existing, []byte("known-good\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	snapshot, err := Take(existing)
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(existing, []byte("candidate\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := snapshot.Restore(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(existing)
	if err != nil || string(data) != "known-good\n" {
		t.Fatalf("restored data = %q, error = %v", data, err)
	}

	created := filepath.Join(root, "created.conf")
	missing, err := Take(created)
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(created, []byte("candidate\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := missing.Restore(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(created); !os.IsNotExist(err) {
		t.Fatalf("new file was not removed: %v", err)
	}
}

func TestTakeRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.WriteFile(target, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Take(link); err == nil {
		t.Fatal("expected symlink error")
	}
}
