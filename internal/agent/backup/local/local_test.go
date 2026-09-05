package local

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/GVALFER/WEBYCP/internal/agent/backup"
	"github.com/GVALFER/WEBYCP/internal/agent/hostuser"
	"github.com/GVALFER/WEBYCP/internal/backupfmt"
)

func TestCreatePreviewAndPartialRestore(t *testing.T) {
	driver := New()
	driver.root = filepath.Join(t.TempDir(), "backups")
	driver.home = filepath.Join(t.TempDir(), "home")
	accountID := "0123456789abcdef0123456789abcdef"
	systemUser := "wcp_0123456789ab"
	home := filepath.Join(driver.home, systemUser)
	driver.lookup = func(string) (*user.User, error) {
		return &user.User{
			Uid: strconv.Itoa(os.Getuid()), Gid: strconv.Itoa(os.Getgid()),
			Name: hostuser.Marker(accountID), HomeDir: home,
		}, nil
	}
	driver.lookupGroup = func(string) (*user.Group, error) {
		return &user.Group{Gid: strconv.Itoa(os.Getgid())}, nil
	}
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
	if err := os.Chmod(filepath.Dir(filePath), 0o750|os.ModeSetgid); err != nil {
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
	if err := os.RemoveAll(filepath.Join(home, "web")); err != nil {
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
	info, err := os.Stat(filepath.Dir(filePath))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSetgid == 0 {
		t.Fatalf("restored directory mode = %v, want setgid", info.Mode())
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

func TestPreviewRejectsIncompleteArchive(t *testing.T) {
	accountID := "0123456789abcdef0123456789abcdef"
	for _, test := range []struct {
		name                          string
		metadata, truncate, duplicate bool
	}{
		{name: "missing metadata", metadata: true},
		{name: "truncated gzip trailer", truncate: true},
		{name: "duplicate manifest", duplicate: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			driver := New()
			driver.root = t.TempDir()
			root := filepath.Join(driver.root, accountID)
			if err := os.Mkdir(root, 0o700); err != nil {
				t.Fatal(err)
			}
			runID := "abcdef0123456789abcdef0123456789"
			filePath := filepath.Join(root, runID+".tar.gz")
			file, err := os.Create(filePath)
			if err != nil {
				t.Fatal(err)
			}
			gz := gzip.NewWriter(file)
			writer := tar.NewWriter(gz)
			value, err := json.Marshal(backupfmt.Manifest{Version: backupfmt.Version, AccountID: accountID, RunID: runID, Metadata: test.metadata})
			if err != nil {
				t.Fatal(err)
			}
			count := 1
			if test.duplicate {
				count = 2
			}
			for range count {
				if err := writer.WriteHeader(&tar.Header{Name: "manifest.json", Mode: 0o600, Size: int64(len(value)), Typeflag: tar.TypeReg}); err != nil {
					t.Fatal(err)
				}
				if _, err := writer.Write(value); err != nil {
					t.Fatal(err)
				}
			}
			if err := errors.Join(writer.Close(), gz.Close(), file.Close()); err != nil {
				t.Fatal(err)
			}
			if test.truncate {
				info, err := os.Stat(filePath)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Truncate(filePath, info.Size()-4); err != nil {
					t.Fatal(err)
				}
			}
			checksum, _, err := fileChecksum(filePath)
			if err != nil {
				t.Fatal(err)
			}
			request := backup.ArtifactRequest{AccountID: accountID, Path: filePath, Checksum: checksum}
			if _, err := driver.Preview(context.Background(), request); err == nil {
				t.Fatal("incomplete archive accepted even with a matching outer checksum")
			}
		})
	}
}

func TestRestoreRejectsMissingScopeBeforeWrites(t *testing.T) {
	driver := New()
	driver.root = t.TempDir()
	accountID := "0123456789abcdef0123456789abcdef"
	artifact, err := driver.Create(context.Background(), backup.CreateRequest{
		RunID: "abcdef0123456789abcdef0123456789", AccountID: accountID,
		SystemUser: "wcp_0123456789ab", Metadata: `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	driver.lookup = func(string) (*user.User, error) {
		t.Fatal("restore reached host identity lookup with an invalid scope")
		return nil, nil
	}
	_, err = driver.Restore(context.Background(), backup.RestoreRequest{
		ArtifactRequest: backup.ArtifactRequest{AccountID: accountID, Path: artifact.Path, Checksum: artifact.Checksum},
		SystemUser:      "wcp_0123456789ab", Files: true,
	})
	if err == nil {
		t.Fatal("missing file scope accepted")
	}
}
