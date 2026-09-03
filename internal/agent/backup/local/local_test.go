package local

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GVALFER/WEBYCP/internal/agent/backup"
)

func TestCreatePreviewAndPartialRestore(t *testing.T) {
	driver := New()
	driver.root = filepath.Join(t.TempDir(), "backups")
	driver.home = filepath.Join(t.TempDir(), "home")
	accountID := "0123456789abcdef0123456789abcdef"
	systemUser := "wcp_0123456789ab"
	home := filepath.Join(driver.home, systemUser)
	database := "wcp_01234567_app"
	driver.dump = func(_ context.Context, writer io.Writer, name string, args ...string) error {
		if name != dumpPath || args[len(args)-1] != database {
			t.Fatalf("unexpected dump command: %s %#v", name, args)
		}
		_, err := io.WriteString(writer, "CREATE DATABASE `"+database+"`;\n")
		return err
	}
	restoredSQL := ""
	driver.input = func(_ context.Context, reader io.Reader, name string, args ...string) error {
		if name != mysqlPath || len(args) != 1 || args[0] != "--protocol=socket" {
			t.Fatalf("unexpected restore command: %s %#v", name, args)
		}
		value, err := io.ReadAll(reader)
		restoredSQL = string(value)
		return err
	}
	filePath := filepath.Join(home, "web", "example.com", "public_html", "index.html")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("before backup"), 0o640); err != nil {
		t.Fatal(err)
	}
	artifact, err := driver.Create(context.Background(), backup.CreateRequest{
		RunID: "abcdef0123456789abcdef0123456789", AccountID: accountID,
		SystemUser: systemUser, IncludeFiles: true, Databases: []string{database},
		Metadata: `{"version":1}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Size == 0 || len(artifact.Manifest.Entries) != 3 {
		t.Fatalf("artifact = %+v", artifact)
	}
	request := backup.ArtifactRequest{AccountID: accountID, Path: artifact.Path, Checksum: artifact.Checksum}
	manifest, err := driver.Preview(context.Background(), request)
	if err != nil || manifest.RunID != "abcdef0123456789abcdef0123456789" {
		t.Fatalf("manifest = %+v, error = %v", manifest, err)
	}
	if err := os.Remove(filePath); err != nil {
		t.Fatal(err)
	}
	metadata, err := driver.Restore(context.Background(), backup.RestoreRequest{ArtifactRequest: request, SystemUser: systemUser, Metadata: true})
	if err != nil || metadata != `{"version":1}` {
		t.Fatalf("metadata = %q, error = %v", metadata, err)
	}
	if _, err := os.Stat(filePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("metadata-only restore wrote files: %v", err)
	}
	metadata, err = driver.Restore(context.Background(), backup.RestoreRequest{
		ArtifactRequest: request, SystemUser: systemUser,
		Files: true, Databases: true, Metadata: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if metadata != `{"version":1}` || !strings.Contains(restoredSQL, "CREATE DATABASE") {
		t.Fatalf("metadata = %q, restored SQL = %q", metadata, restoredSQL)
	}
	data, err := os.ReadFile(filePath)
	if err != nil || string(data) != "before backup" {
		t.Fatalf("restored data = %q, error = %v", data, err)
	}
}

func TestPreviewRejectsTamperedArtifact(t *testing.T) {
	driver := New()
	driver.root = filepath.Join(t.TempDir(), "backups")
	driver.home = filepath.Join(t.TempDir(), "home")
	accountID := "0123456789abcdef0123456789abcdef"
	systemUser := "wcp_0123456789ab"
	if err := os.MkdirAll(filepath.Join(driver.home, systemUser), 0o750); err != nil {
		t.Fatal(err)
	}
	artifact, err := driver.Create(context.Background(), backup.CreateRequest{RunID: "abcdef0123456789abcdef0123456789", AccountID: accountID, SystemUser: systemUser, Metadata: `{}`})
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(artifact.Path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("tampered")); err != nil {
		t.Fatal(err)
	}
	file.Close()
	_, err = driver.Preview(context.Background(), backup.ArtifactRequest{AccountID: accountID, Path: artifact.Path, Checksum: artifact.Checksum})
	if err == nil {
		t.Fatal("expected checksum error")
	}
}

func TestSafeTargetRejectsExistingSymlink(t *testing.T) {
	root := t.TempDir()
	if err := os.Symlink(t.TempDir(), filepath.Join(root, "web")); err != nil {
		t.Fatal(err)
	}
	if _, err := safeTarget(root, "web/index.html"); err == nil {
		t.Fatal("expected symlink target to be rejected")
	}
}
