package packages

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/GVALFER/WEBYCP/internal/pagination"
)

var (
	ErrNameExists = errors.New("package name already exists")
	ErrInUse      = errors.New("package is assigned to accounts")
)

const DefaultID = "00000000000000000000000000000001"

type Resource string

const (
	Websites       Resource = "websites"
	Domains        Resource = "domains"
	Aliases        Resource = "aliases"
	Databases      Resource = "databases"
	DatabaseUsers  Resource = "database users"
	ScheduledTasks Resource = "scheduled tasks"
	BackupPlans    Resource = "backup plans"
	FTPAccounts    Resource = "FTP accounts"
)

type LimitError struct {
	Resource Resource
	Limit    int64
}

func (e *LimitError) Error() string {
	return fmt.Sprintf("%s limit of %d reached", e.Resource, e.Limit)
}

type Limits struct {
	Websites, Domains, Aliases               int64
	Databases, DatabaseUsers, ScheduledTasks int64
	BackupPlans, BackupRetention             int64
	FTPAccounts                              int64
}

type Usage struct {
	Websites, Domains, Aliases               int64
	Databases, DatabaseUsers, ScheduledTasks int64
	BackupPlans                              int64
	FTPAccounts                              int64
}

type Package struct {
	ID, Name             string
	Limits               Limits
	AccountCount         int64
	CreatedAt, UpdatedAt time.Time
}

type Repository interface {
	CreatePackage(context.Context, Package) (Package, error)
	Package(context.Context, string) (Package, error)
	PackagePage(context.Context, pagination.Query) (pagination.Result[Package], error)
	UpdatePackage(context.Context, Package) (Package, error)
	DeletePackage(context.Context, string) error
	AssignPackage(context.Context, string, string, time.Time) error
}
