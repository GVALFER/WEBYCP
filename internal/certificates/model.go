package certificates

import (
	"context"
	"errors"
	"time"

	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/pagination"
)

var ErrBusy = errors.New("certificate operation is pending")

type Certificate struct {
	ID, DomainID, NodeID, Kind, Name, Email, Status, Error string
	Names                                                  []string
	RedirectHTTPS                                          bool
	ExpiresAt, RenewAfter                                  *time.Time
	CreatedAt, UpdatedAt                                   time.Time
}

type Request struct {
	CertificateID, Kind, DomainID, AccountID, SystemUser string
	Name, Email, PHPVersion                              string
	Names                                                []string
	RedirectHTTPS                                        bool
}

type Result struct {
	Names     []string
	ExpiresAt time.Time
}

type Repository interface {
	Certificates(context.Context, string, bool) ([]Certificate, error)
	CertificatePage(context.Context, string, bool, pagination.Query) (pagination.Result[Certificate], error)
	Certificate(context.Context, string) (Certificate, error)
	DomainCertificate(context.Context, string) (Certificate, error)
	PanelCertificate(context.Context) (Certificate, error)
	QueueCertificate(context.Context, Certificate, jobs.Job, bool) (Certificate, jobs.Job, error)
	SetResult(context.Context, string, []string, string, *time.Time, *time.Time, string) error
	DueCertificates(context.Context, time.Time) ([]Certificate, error)
	CertificateJobPending(context.Context, string) (bool, error)
}

type Agent interface {
	IssueCertificate(context.Context, string, Request) (Result, error)
}
