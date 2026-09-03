package fsx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDirectory(t *testing.T) {
	rootPath := t.TempDir()
	root, err := OpenDir(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	if err := root.Ensure("web", 0o750, os.Getuid(), os.Getgid()); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(rootPath, "web"))
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o750 {
		t.Fatalf("mode = %o, want 750", mode)
	}
}

func TestOpenRejectsSymlink(t *testing.T) {
	rootPath := t.TempDir()
	if err := os.Symlink(t.TempDir(), filepath.Join(rootPath, "web")); err != nil {
		t.Fatal(err)
	}
	root, err := OpenDir(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	if err := root.Ensure("web", 0o750, os.Getuid(), os.Getgid()); err == nil {
		t.Fatal("expected symlink error")
	}
}

func TestEnsureRejectsNestedName(t *testing.T) {
	root, err := OpenDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.Ensure("../outside", 0o750, os.Getuid(), os.Getgid()); err == nil {
		t.Fatal("expected invalid name error")
	}
}

func TestRenameDirectory(t *testing.T) {
	rootPath := t.TempDir()
	if err := os.Mkdir(filepath.Join(rootPath, "source"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(rootPath, "target"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(rootPath, "source", "site"), 0o750); err != nil {
		t.Fatal(err)
	}
	root, err := OpenDir(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	source, err := root.Open("source")
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	target, err := root.Open("target")
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()

	if err := source.Rename("site", target, "domain-id"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(rootPath, "target", "domain-id")); err != nil {
		t.Fatal(err)
	}
}

func TestRenameDoesNotReplaceTarget(t *testing.T) {
	rootPath := t.TempDir()
	for _, path := range []string{"source", "target", "source/site", "target/domain-id"} {
		if err := os.Mkdir(filepath.Join(rootPath, path), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	root, err := OpenDir(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	source, err := root.Open("source")
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	target, err := root.Open("target")
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()

	if err := source.Rename("site", target, "domain-id"); err == nil {
		t.Fatal("expected existing target error")
	}
}
