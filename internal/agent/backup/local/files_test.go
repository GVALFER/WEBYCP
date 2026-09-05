package local

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GVALFER/WEBYCP/internal/agent/hostuser"
	"github.com/GVALFER/WEBYCP/internal/backupfmt"
)

func TestRestoreReplacesHardLink(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("untouched"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(outside, filepath.Join(dir, "file.txt")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	identity := hostuser.Identity{UID: os.Getuid(), GID: os.Getgid(), Home: dir}
	header := &tar.Header{Name: "files/file.txt", Typeflag: tar.TypeReg, Size: 8, Mode: 0o640}
	if err := New().restoreFile(root, strings.NewReader("restored"), header, identity, os.Getgid()); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(outside); err != nil || string(data) != "untouched" {
		t.Fatalf("hard-linked outside file changed: %q, %v", data, err)
	}
	if data, err := root.ReadFile("file.txt"); err != nil || string(data) != "restored" {
		t.Fatalf("restored file = %q, %v", data, err)
	}
	if err := New().restoreFile(root, strings.NewReader("short"), header, identity, os.Getgid()); err == nil {
		t.Fatal("partial write accepted")
	}
	if data, err := root.ReadFile("file.txt"); err != nil || string(data) != "restored" {
		t.Fatalf("failed restore replaced existing content: %q, %v", data, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary restore files remain: %v, %v", entries, err)
	}
}

func TestBackupAndRestoreStayInOpenedHome(t *testing.T) {
	base := t.TempDir()
	home, moved, outside := filepath.Join(base, "home"), filepath.Join(base, "moved"), t.TempDir()
	if err := os.Mkdir(home, 0o750); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(home)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := os.Rename(home, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, home); err != nil {
		t.Fatal(err)
	}
	identity := hostuser.Identity{UID: os.Getuid(), GID: os.Getgid(), Home: home}
	header := &tar.Header{Name: "files/file.txt", Typeflag: tar.TypeReg, Size: 4, Mode: 0o600}
	if err := New().restoreFile(root, strings.NewReader("safe"), header, identity, os.Getgid()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(outside, "file.txt")); !os.IsNotExist(err) {
		t.Fatalf("restore escaped the opened home: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(moved, "file.txt")); err != nil || string(data) != "safe" {
		t.Fatalf("restore did not use the pinned home: %q, %v", data, err)
	}
	if err := os.Symlink(outside, filepath.Join(moved, "link")); err != nil {
		t.Fatal(err)
	}
	writer := tar.NewWriter(io.Discard)
	defer writer.Close()
	manifest := backupfmt.Manifest{}
	if err := addPath(writer, root, "file.txt", "files/file.txt", &manifest); err != nil || len(manifest.Entries) != 1 {
		t.Fatalf("backup did not use the pinned home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outside, "private.txt"), []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := addPath(writer, root, "link/private.txt", "files/private.txt", &manifest); err == nil {
		t.Fatal("backup read outside the opened home")
	}
	header.Name = "files/link/private.txt"
	if err := New().restoreFile(root, strings.NewReader("safe"), header, identity, os.Getgid()); err == nil {
		t.Fatal("restore followed an outside symlink")
	}
}
