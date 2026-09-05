package ftp

import (
	"context"
	"errors"
	"time"

	agentftp "github.com/GVALFER/WEBYCP/internal/agent/ftp"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/pagination"
)

var (
	ErrNameExists = errors.New("FTP username already exists on this node")
	ErrBusy       = errors.New("an FTP change is already queued for this account")
	ErrDeleting   = errors.New("FTP account is awaiting deletion")
)

type Account struct {
	ID, AccountID, NodeID, Username, Status string
	AccountName, AccountStatus, SystemUser  string
	Enabled, Deleting                       bool
	CreatedAt, UpdatedAt                    time.Time
}

// Credentials never cross the public API, audit or Job payload boundary.
type Credential struct {
	Account
	PasswordHash string `json:"-"`
}

type Changes struct {
	Username, PasswordHash *string
	Enabled                *bool
	Deleting               bool
}

type Repository interface {
	CreateFTP(context.Context, Credential, jobs.Job) (Account, jobs.Job, error)
	ChangeFTP(context.Context, string, Changes, jobs.Job) (Account, jobs.Job, error)
	FTP(context.Context, string) (Credential, error)
	FTPPage(context.Context, string, bool, pagination.Query) (pagination.Result[Account], error)
	AccountFTP(context.Context, string) ([]Credential, error)
	FinishFTP(context.Context, string, bool) error
}

type Agent interface {
	SyncFTP(context.Context, string, string, string, []agentftp.Entry) error
}
