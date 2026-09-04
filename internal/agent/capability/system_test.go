package capability

import (
	"context"
	"errors"
	"io/fs"
	"testing"
)

type fileInfo struct {
	fs.FileInfo
	directory bool
}

func (f fileInfo) IsDir() bool { return f.directory }

func TestObserve(t *testing.T) {
	system := &System{
		run: func(_ context.Context, name string, _ ...string) error {
			if name == "mysqladmin" {
				return errors.New("mysql is unavailable")
			}
			return nil
		},
		stat: func(string) (fs.FileInfo, error) { return fileInfo{directory: true}, nil },
	}

	value := system.Observe(context.Background())
	if value.Webservers[0].Status != "healthy" ||
		value.Runtimes[0].Version != "8.3" ||
		value.Databases[0].Status != "unavailable" ||
		value.Schedulers[0].Status != "healthy" ||
		value.Backups[0].Status != "healthy" {
		t.Fatalf("capabilities = %+v", value)
	}
}

func TestObserveMissingBackupDirectory(t *testing.T) {
	system := &System{
		run:  func(context.Context, string, ...string) error { return nil },
		stat: func(string) (fs.FileInfo, error) { return nil, fs.ErrNotExist },
	}
	if status := system.Observe(context.Background()).Backups[0].Status; status != "unavailable" {
		t.Fatalf("backup status = %q", status)
	}
}
