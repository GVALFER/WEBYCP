package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	agentbackup "github.com/GVALFER/WEBYCP/internal/agent/backup"
	agentserver "github.com/GVALFER/WEBYCP/internal/agent/server"
	"github.com/GVALFER/WEBYCP/internal/backupfmt"
)

type backupFailure struct{ err error }

func (d backupFailure) Create(context.Context, agentbackup.CreateRequest) (agentbackup.Artifact, error) {
	return agentbackup.Artifact{}, d.err
}

func (d backupFailure) Preview(context.Context, agentbackup.ArtifactRequest) (backupfmt.Manifest, error) {
	return backupfmt.Manifest{}, d.err
}

func (d backupFailure) Restore(context.Context, agentbackup.RestoreRequest) (string, error) {
	return "", d.err
}

func (d backupFailure) Delete(context.Context, agentbackup.ArtifactRequest) error { return d.err }

func TestBackupErrors(t *testing.T) {
	for name, failure := range map[string]error{"version": backupfmt.ErrVersion, "invalid": backupfmt.ErrInvalid, "internal": errors.New("private storage path")} {
		t.Run(name, func(t *testing.T) {
			socket, server := testServer(t, agentserver.Options{Backups: backupFailure{err: fmt.Errorf("private detail: %w", failure)}})
			defer server.Shutdown(context.Background())
			client := New(time.Second)
			_, preview := client.PreviewBackup(context.Background(), socket, agentbackup.ArtifactRequest{})
			_, restore := client.RestoreBackup(context.Background(), socket, agentbackup.RestoreRequest{})
			for _, err := range []error{preview, restore} {
				if err == nil || strings.Contains(err.Error(), "private") {
					t.Fatalf("unexpected public error: %v", err)
				}
				if failure == backupfmt.ErrVersion || failure == backupfmt.ErrInvalid {
					if !errors.Is(err, failure) {
						t.Fatalf("error = %v, want %v", err, failure)
					}
				} else if errors.Is(err, backupfmt.ErrVersion) || errors.Is(err, backupfmt.ErrInvalid) {
					t.Fatalf("internal failure classified as invalid archive: %v", err)
				}
			}
		})
	}
}
