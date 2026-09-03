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

type Driver interface {
	Create(context.Context, CreateRequest) (Artifact, error)
	Preview(context.Context, ArtifactRequest) (backupfmt.Manifest, error)
	Restore(context.Context, RestoreRequest) (string, error)
	Delete(context.Context, ArtifactRequest) error
}
