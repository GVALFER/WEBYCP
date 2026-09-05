package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/GVALFER/WEBYCP/internal/packages"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite/dbgen"
)

func (s *Store) CreatePackage(ctx context.Context, value packages.Package) (packages.Package, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return packages.Package{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	exists, err := q.PackageNameExists(ctx, dbgen.PackageNameExistsParams{Name: value.Name, ID: value.ID})
	if err != nil {
		return packages.Package{}, err
	}
	if exists {
		return packages.Package{}, packages.ErrNameExists
	}
	row, err := q.CreatePackage(ctx, createPackageParams(value))
	if err != nil {
		return packages.Package{}, err
	}
	if err := tx.Commit(); err != nil {
		return packages.Package{}, err
	}
	return packageValue(row, 0), nil
}

func (s *Store) Package(ctx context.Context, id string) (packages.Package, error) {
	row, err := s.queries.GetPackageOverview(ctx, id)
	if err != nil {
		return packages.Package{}, err
	}
	return packageOverviewValue(row), nil
}

func (s *Store) PackagePage(ctx context.Context, query pagination.Query) (pagination.Result[packages.Package], error) {
	total, err := s.queries.CountPackages(ctx)
	if err != nil {
		return pagination.Result[packages.Package]{}, err
	}
	query = pagination.Clamp(query, total)
	rows, err := s.queries.ListPackageOverviewsPage(ctx, dbgen.ListPackageOverviewsPageParams{
		PageSize: int64(query.Size), PageOffset: pagination.Offset(query),
	})
	if err != nil {
		return pagination.Result[packages.Package]{}, err
	}
	items := make([]packages.Package, 0, len(rows))
	for _, row := range rows {
		items = append(items, packageOverviewValue(row))
	}
	return pagination.Result[packages.Package]{Items: items, Query: query, Total: total}, nil
}

func (s *Store) UpdatePackage(ctx context.Context, value packages.Package) (packages.Package, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return packages.Package{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	exists, err := q.PackageNameExists(ctx, dbgen.PackageNameExistsParams{Name: value.Name, ID: value.ID})
	if err != nil {
		return packages.Package{}, err
	}
	if exists {
		return packages.Package{}, packages.ErrNameExists
	}
	row, err := q.UpdatePackage(ctx, dbgen.UpdatePackageParams{
		Name: value.Name, MaxWebsites: value.Limits.Websites,
		MaxDomains: value.Limits.Domains, MaxAliases: value.Limits.Aliases,
		MaxDatabases: value.Limits.Databases, MaxDatabaseUsers: value.Limits.DatabaseUsers,
		MaxScheduledTasks: value.Limits.ScheduledTasks, MaxBackupPlans: value.Limits.BackupPlans,
		MaxBackupRetention: value.Limits.BackupRetention, UpdatedAt: timeValue(value.UpdatedAt), ID: value.ID,
		MaxFtpAccounts: value.Limits.FTPAccounts,
	})
	if err != nil {
		return packages.Package{}, err
	}
	if err := tx.Commit(); err != nil {
		return packages.Package{}, err
	}
	return packageValue(row, value.AccountCount), nil
}

func (s *Store) DeletePackage(ctx context.Context, id string) error {
	count, err := s.queries.DeletePackage(ctx, id)
	if err != nil {
		return err
	}
	if count == 0 {
		if _, err := s.queries.GetPackageOverview(ctx, id); err != nil {
			return err
		}
		return packages.ErrInUse
	}
	return nil
}

func (s *Store) AssignPackage(ctx context.Context, accountID, packageID string, now time.Time) error {
	return s.queries.AssignAccountPackage(ctx, dbgen.AssignAccountPackageParams{
		AccountID: accountID, PackageID: packageID,
		CreatedAt: timeValue(now), UpdatedAt: timeValue(now),
	})
}

func createPackageParams(value packages.Package) dbgen.CreatePackageParams {
	return dbgen.CreatePackageParams{
		ID: value.ID, Name: value.Name, MaxWebsites: value.Limits.Websites,
		MaxDomains: value.Limits.Domains, MaxAliases: value.Limits.Aliases,
		MaxDatabases: value.Limits.Databases, MaxDatabaseUsers: value.Limits.DatabaseUsers,
		MaxScheduledTasks: value.Limits.ScheduledTasks, MaxBackupPlans: value.Limits.BackupPlans,
		MaxBackupRetention: value.Limits.BackupRetention,
		MaxFtpAccounts:     value.Limits.FTPAccounts,
		CreatedAt:          timeValue(value.CreatedAt), UpdatedAt: timeValue(value.UpdatedAt),
	}
}

func packageValue(row dbgen.Package, accountCount int64) packages.Package {
	return packages.Package{
		ID: row.ID, Name: row.Name,
		Limits: packages.Limits{
			Websites: row.MaxWebsites, Domains: row.MaxDomains, Aliases: row.MaxAliases,
			Databases: row.MaxDatabases, DatabaseUsers: row.MaxDatabaseUsers,
			ScheduledTasks: row.MaxScheduledTasks, BackupPlans: row.MaxBackupPlans,
			BackupRetention: row.MaxBackupRetention,
			FTPAccounts:     row.MaxFtpAccounts,
		},
		AccountCount: accountCount, CreatedAt: timeFrom(row.CreatedAt), UpdatedAt: timeFrom(row.UpdatedAt),
	}
}

func packageOverviewValue(row dbgen.PackageOverview) packages.Package {
	return packages.Package{
		ID: row.ID, Name: row.Name,
		Limits: packages.Limits{
			Websites: row.MaxWebsites, Domains: row.MaxDomains, Aliases: row.MaxAliases,
			Databases: row.MaxDatabases, DatabaseUsers: row.MaxDatabaseUsers,
			ScheduledTasks: row.MaxScheduledTasks, BackupPlans: row.MaxBackupPlans,
			BackupRetention: row.MaxBackupRetention,
			FTPAccounts:     row.MaxFtpAccounts,
		},
		AccountCount: row.AccountCount, CreatedAt: timeFrom(row.CreatedAt), UpdatedAt: timeFrom(row.UpdatedAt),
	}
}

type limitKind int

const (
	limitWebsites limitKind = iota
	limitDomains
	limitAliases
	limitDatabases
	limitDatabaseUsers
	limitScheduledTasks
	limitBackupPlans
	limitFTPAccounts
)

func requireCapacity(ctx context.Context, tx *sql.Tx, accountID string, kind limitKind) error {
	queries := [...]string{
		`SELECT packages.max_websites, (SELECT COUNT(*) FROM websites WHERE websites.account_id = accounts.id) FROM accounts JOIN account_packages ON account_packages.account_id = accounts.id JOIN packages ON packages.id = account_packages.package_id WHERE accounts.id = ?`,
		`SELECT packages.max_domains, (SELECT COUNT(*) FROM website_domains JOIN websites ON websites.id = website_domains.website_id WHERE websites.account_id = accounts.id AND website_domains.kind = 'primary') FROM accounts JOIN account_packages ON account_packages.account_id = accounts.id JOIN packages ON packages.id = account_packages.package_id WHERE accounts.id = ?`,
		`SELECT packages.max_aliases, (SELECT COUNT(*) FROM website_domains JOIN websites ON websites.id = website_domains.website_id WHERE websites.account_id = accounts.id AND website_domains.kind = 'alias') FROM accounts JOIN account_packages ON account_packages.account_id = accounts.id JOIN packages ON packages.id = account_packages.package_id WHERE accounts.id = ?`,
		`SELECT packages.max_databases, (SELECT COUNT(*) FROM databases WHERE databases.account_id = accounts.id) FROM accounts JOIN account_packages ON account_packages.account_id = accounts.id JOIN packages ON packages.id = account_packages.package_id WHERE accounts.id = ?`,
		`SELECT packages.max_database_users, (SELECT COUNT(*) FROM database_users WHERE database_users.account_id = accounts.id) FROM accounts JOIN account_packages ON account_packages.account_id = accounts.id JOIN packages ON packages.id = account_packages.package_id WHERE accounts.id = ?`,
		`SELECT packages.max_scheduled_tasks, (SELECT COUNT(*) FROM scheduled_tasks WHERE scheduled_tasks.account_id = accounts.id) FROM accounts JOIN account_packages ON account_packages.account_id = accounts.id JOIN packages ON packages.id = account_packages.package_id WHERE accounts.id = ?`,
		`SELECT packages.max_backup_plans, (SELECT COUNT(*) FROM backup_plans WHERE backup_plans.account_id = accounts.id) FROM accounts JOIN account_packages ON account_packages.account_id = accounts.id JOIN packages ON packages.id = account_packages.package_id WHERE accounts.id = ?`,
		`SELECT packages.max_ftp_accounts, (SELECT COUNT(*) FROM ftp_accounts WHERE ftp_accounts.account_id = accounts.id) FROM accounts JOIN account_packages ON account_packages.account_id = accounts.id JOIN packages ON packages.id = account_packages.package_id WHERE accounts.id = ?`,
	}
	resources := [...]packages.Resource{
		packages.Websites, packages.Domains, packages.Aliases, packages.Databases,
		packages.DatabaseUsers, packages.ScheduledTasks, packages.BackupPlans, packages.FTPAccounts,
	}
	if kind < limitWebsites || kind > limitFTPAccounts {
		return fmt.Errorf("unknown package limit")
	}
	var limit, used int64
	if err := tx.QueryRowContext(ctx, queries[kind], accountID).Scan(&limit, &used); err != nil {
		return err
	}
	if used >= limit {
		return &packages.LimitError{Resource: resources[kind], Limit: limit}
	}
	return nil
}

func requireRetention(ctx context.Context, tx *sql.Tx, accountID string, requested, current int64) error {
	var limit int64
	err := tx.QueryRowContext(ctx, `
		SELECT packages.max_backup_retention
		FROM account_packages
		JOIN packages ON packages.id = account_packages.package_id
		WHERE account_packages.account_id = ?
	`, accountID).Scan(&limit)
	if err != nil {
		return err
	}
	if requested > limit && requested > current {
		return &packages.LimitError{Resource: "backup retention", Limit: limit}
	}
	return nil
}
