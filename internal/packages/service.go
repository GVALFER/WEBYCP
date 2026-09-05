package packages

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GVALFER/WEBYCP/internal/idgen"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, value Package) (Package, error) {
	if err := prepare(&value); err != nil {
		return Package{}, err
	}
	id, err := idgen.ID()
	if err != nil {
		return Package{}, err
	}
	now := time.Now().UTC()
	value.ID, value.CreatedAt, value.UpdatedAt = id, now, now
	return s.repository.CreatePackage(ctx, value)
}

func (s *Service) Update(ctx context.Context, id string, value Package) (Package, error) {
	if err := validate.ID("packageId", id); err != nil {
		return Package{}, err
	}
	current, err := s.repository.Package(ctx, id)
	if err != nil {
		return Package{}, err
	}
	if err := prepare(&value); err != nil {
		return Package{}, err
	}
	value.ID, value.CreatedAt, value.UpdatedAt = id, current.CreatedAt, time.Now().UTC()
	value.AccountCount = current.AccountCount
	return s.repository.UpdatePackage(ctx, value)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := validate.ID("packageId", id); err != nil {
		return err
	}
	return s.repository.DeletePackage(ctx, id)
}

func (s *Service) Package(ctx context.Context, id string) (Package, error) {
	if err := validate.ID("packageId", id); err != nil {
		return Package{}, err
	}
	return s.repository.Package(ctx, id)
}

func (s *Service) Page(ctx context.Context, query pagination.Query) (pagination.Result[Package], error) {
	return s.repository.PackagePage(ctx, query)
}

func (s *Service) Assign(ctx context.Context, accountID, packageID string) error {
	if err := validate.ID("accountId", accountID); err != nil {
		return err
	}
	if _, err := s.Package(ctx, packageID); err != nil {
		return err
	}
	return s.repository.AssignPackage(ctx, accountID, packageID, time.Now().UTC())
}

func prepare(value *Package) error {
	value.Name = strings.TrimSpace(value.Name)
	if len(value.Name) < 2 || len(value.Name) > 80 {
		return &validate.Error{Field: "name", Message: "Use a name between 2 and 80 characters"}
	}
	limits := []struct {
		field string
		value int64
	}{
		{"maxWebsites", value.Limits.Websites},
		{"maxDomains", value.Limits.Domains},
		{"maxAliases", value.Limits.Aliases},
		{"maxDatabases", value.Limits.Databases},
		{"maxDatabaseUsers", value.Limits.DatabaseUsers},
		{"maxScheduledTasks", value.Limits.ScheduledTasks},
		{"maxBackupPlans", value.Limits.BackupPlans},
	}
	for _, limit := range limits {
		if limit.value < 0 || limit.value > 1_000_000 {
			return &validate.Error{Field: limit.field, Message: "Use a limit between 0 and 1000000"}
		}
	}
	if value.Limits.BackupRetention < 1 || value.Limits.BackupRetention > 100 {
		return &validate.Error{Field: "maxBackupRetention", Message: "Use a retention between 1 and 100"}
	}
	if value.Limits.FTPAccounts < 0 || value.Limits.FTPAccounts > 100 {
		return &validate.Error{Field: "maxFTPAccounts", Message: "Use an FTP account limit between 0 and 100"}
	}
	if value.Limits.Domains < value.Limits.Websites {
		return &validate.Error{Field: "maxDomains", Message: fmt.Sprintf("Use at least %d domains for %d websites", value.Limits.Websites, value.Limits.Websites)}
	}
	return nil
}
