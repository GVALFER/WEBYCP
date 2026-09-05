package backup

import (
	"context"

	"github.com/GVALFER/WEBYCP/internal/backupfmt"
)

type CreateRequest struct {
	RunID, AccountID, SystemUser, Metadata string
	IncludeFiles                           bool
	Databases                              []string
}

type Artifact struct {
	Path, Checksum string
	Size           int64
	Manifest       backupfmt.Manifest
}

type ArtifactRequest struct {
	AccountID, Path, Checksum string
}

type RestoreRequest struct {
	ArtifactRequest
	SystemUser                 string
	Files, Databases, Metadata bool
}

// Driver owns artifact storage and restores on the account's node. Preview and
// Restore must verify archive identity, paths and checksums before any writes.
// Restore must honor the selected scope and repair managed file ownership.
// Delete is idempotent; a missing artifact is already deleted.
type Driver interface {
	Create(context.Context, CreateRequest) (Artifact, error)
	Preview(context.Context, ArtifactRequest) (backupfmt.Manifest, error)
	Restore(context.Context, RestoreRequest) (string, error)
	Delete(context.Context, ArtifactRequest) error
}
