package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite/dbgen"
	"github.com/GVALFER/WEBYCP/internal/websites"
)

func (s *Store) CreateWebsiteProvision(ctx context.Context, website websites.Website, domain websites.WebsiteDomain, job jobs.Job) (websites.Website, websites.WebsiteDomain, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return websites.Website{}, websites.WebsiteDomain{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	if err := requireCapacity(ctx, tx, website.AccountID, limitWebsites); err != nil {
		return websites.Website{}, websites.WebsiteDomain{}, jobs.Job{}, err
	}
	if err := requireCapacity(ctx, tx, website.AccountID, limitDomains); err != nil {
		return websites.Website{}, websites.WebsiteDomain{}, jobs.Job{}, err
	}
	if err := ensureHostnameAvailable(ctx, q, domain.Hostname); err != nil {
		return websites.Website{}, websites.WebsiteDomain{}, jobs.Job{}, err
	}
	createdWebsite, err := q.CreateWebsite(ctx, dbgen.CreateWebsiteParams{
		ID: website.ID, AccountID: website.AccountID, NodeID: website.NodeID,
		Name: website.Name, Kind: website.Kind, DocumentRoot: website.DocumentRoot,
		WebDriver: website.WebDriver, RuntimeDriver: website.RuntimeDriver,
		RuntimeVersion: website.RuntimeVersion, CreatedAt: timeValue(website.CreatedAt),
		UpdatedAt: timeValue(website.UpdatedAt),
	})
	if err != nil {
		return websites.Website{}, websites.WebsiteDomain{}, jobs.Job{}, fmt.Errorf("insert website: %w", err)
	}
	createdDomain, err := q.CreateWebsiteDomain(ctx, dbgen.CreateWebsiteDomainParams{
		ID: domain.ID, WebsiteID: domain.WebsiteID, Hostname: domain.Hostname, Kind: domain.Kind,
		CreatedAt: timeValue(domain.CreatedAt), UpdatedAt: timeValue(domain.UpdatedAt),
	})
	if err != nil {
		return websites.Website{}, websites.WebsiteDomain{}, jobs.Job{}, fmt.Errorf("insert primary domain: %w", err)
	}
	createdJob, err := createJob(ctx, q, job)
	if err != nil {
		return websites.Website{}, websites.WebsiteDomain{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return websites.Website{}, websites.WebsiteDomain{}, jobs.Job{}, err
	}
	return websiteValue(createdWebsite), websiteDomainValue(createdDomain), createdJob, nil
}

func (s *Store) CreateWebsiteDomainProvision(ctx context.Context, domain websites.WebsiteDomain, job jobs.Job) (websites.WebsiteDomain, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return websites.WebsiteDomain{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	website, err := q.GetWebsite(ctx, domain.WebsiteID)
	if err != nil {
		return websites.WebsiteDomain{}, jobs.Job{}, err
	}
	if err := requireCapacity(ctx, tx, website.AccountID, limitAliases); err != nil {
		return websites.WebsiteDomain{}, jobs.Job{}, err
	}
	if err := ensureHostnameAvailable(ctx, q, domain.Hostname); err != nil {
		return websites.WebsiteDomain{}, jobs.Job{}, err
	}
	created, err := q.CreateWebsiteDomain(ctx, dbgen.CreateWebsiteDomainParams{
		ID: domain.ID, WebsiteID: domain.WebsiteID, Hostname: domain.Hostname, Kind: domain.Kind,
		CreatedAt: timeValue(domain.CreatedAt), UpdatedAt: timeValue(domain.UpdatedAt),
	})
	if err != nil {
		return websites.WebsiteDomain{}, jobs.Job{}, fmt.Errorf("insert website domain: %w", err)
	}
	createdJob, err := createJob(ctx, q, job)
	if err != nil {
		return websites.WebsiteDomain{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return websites.WebsiteDomain{}, jobs.Job{}, err
	}
	return websiteDomainValue(created), createdJob, nil
}

func (s *Store) Website(ctx context.Context, id string) (websites.Website, error) {
	row, err := s.queries.GetWebsite(ctx, id)
	if err != nil {
		return websites.Website{}, err
	}
	return websiteValue(row), nil
}

func (s *Store) Websites(ctx context.Context, userID string, admin bool) ([]websites.Website, error) {
	var rows []dbgen.Website
	var err error
	if admin {
		rows, err = s.queries.ListWebsites(ctx)
	} else {
		rows, err = s.queries.ListUserWebsites(ctx, userID)
	}
	if err != nil {
		return nil, err
	}
	return websiteValues(rows), nil
}

func (s *Store) WebsitePage(ctx context.Context, userID string, admin bool, query pagination.Query) (pagination.Result[websites.Website], error) {
	var total int64
	var rows []dbgen.Website
	var err error
	if admin {
		total, err = s.queries.CountWebsites(ctx)
	} else {
		total, err = s.queries.CountUserWebsites(ctx, userID)
	}
	if err != nil {
		return pagination.Result[websites.Website]{}, err
	}
	query = pagination.Clamp(query, total)
	if admin {
		rows, err = s.queries.ListWebsitesPage(ctx, dbgen.ListWebsitesPageParams{PageSize: int64(query.Size), PageOffset: pagination.Offset(query)})
	} else {
		rows, err = s.queries.ListUserWebsitesPage(ctx, dbgen.ListUserWebsitesPageParams{UserID: userID, PageSize: int64(query.Size), PageOffset: pagination.Offset(query)})
	}
	if err != nil {
		return pagination.Result[websites.Website]{}, err
	}
	return pagination.Result[websites.Website]{Items: websiteValues(rows), Query: query, Total: total}, nil
}

func (s *Store) WebsiteDomain(ctx context.Context, id string) (websites.WebsiteDomain, error) {
	row, err := s.queries.GetWebsiteDomain(ctx, id)
	if err != nil {
		return websites.WebsiteDomain{}, err
	}
	return websiteDomainValue(row), nil
}

func (s *Store) PrimaryDomain(ctx context.Context, websiteID string) (websites.WebsiteDomain, error) {
	row, err := s.queries.GetWebsitePrimaryDomain(ctx, websiteID)
	if err != nil {
		return websites.WebsiteDomain{}, err
	}
	return websiteDomainValue(row), nil
}

func (s *Store) WebsiteDomains(ctx context.Context, websiteID string) ([]websites.WebsiteDomain, error) {
	rows, err := s.queries.ListWebsiteDomains(ctx, websiteID)
	if err != nil {
		return nil, err
	}
	return websiteDomainValues(rows), nil
}

func (s *Store) EnabledWebsiteDomains(ctx context.Context, websiteID string) ([]websites.WebsiteDomain, error) {
	rows, err := s.queries.ListEnabledWebsiteDomains(ctx, websiteID)
	if err != nil {
		return nil, err
	}
	return websiteDomainValues(rows), nil
}

func (s *Store) WebsiteDomainPage(ctx context.Context, userID string, admin bool, kind string, query pagination.Query) (pagination.Result[websites.WebsiteDomain], error) {
	args := dbgen.CountWebsiteDomainsByKindParams{Kind: kind, IsAdmin: admin, UserID: userID}
	total, err := s.queries.CountWebsiteDomainsByKind(ctx, args)
	if err != nil {
		return pagination.Result[websites.WebsiteDomain]{}, err
	}
	query = pagination.Clamp(query, total)
	rows, err := s.queries.ListWebsiteDomainsByKindPage(ctx, dbgen.ListWebsiteDomainsByKindPageParams{Kind: kind, IsAdmin: admin, UserID: userID, PageSize: int64(query.Size), PageOffset: pagination.Offset(query)})
	if err != nil {
		return pagination.Result[websites.WebsiteDomain]{}, err
	}
	return pagination.Result[websites.WebsiteDomain]{Items: websiteDomainValues(rows), Query: query, Total: total}, nil
}

func (s *Store) QueueWebsiteAction(ctx context.Context, id string, enabled bool, job jobs.Job) (websites.Website, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return websites.Website{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	row, err := q.QueueWebsiteAction(ctx, dbgen.QueueWebsiteActionParams{Enabled: boolValue(enabled), UpdatedAt: timeValue(time.Now().UTC()), ID: id})
	if errors.Is(err, sql.ErrNoRows) {
		return websites.Website{}, jobs.Job{}, websites.ErrWebsiteBusy
	}
	if err != nil {
		return websites.Website{}, jobs.Job{}, err
	}
	createdJob, err := createJob(ctx, q, job)
	if err != nil {
		return websites.Website{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return websites.Website{}, jobs.Job{}, err
	}
	return websiteValue(row), createdJob, nil
}

func (s *Store) QueueWebsiteDomainAction(ctx context.Context, id string, enabled bool, job jobs.Job) (websites.WebsiteDomain, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return websites.WebsiteDomain{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	row, err := q.QueueWebsiteDomainAction(ctx, dbgen.QueueWebsiteDomainActionParams{Enabled: boolValue(enabled), UpdatedAt: timeValue(time.Now().UTC()), ID: id})
	if errors.Is(err, sql.ErrNoRows) {
		return websites.WebsiteDomain{}, jobs.Job{}, websites.ErrWebsiteDomainBusy
	}
	if err != nil {
		return websites.WebsiteDomain{}, jobs.Job{}, err
	}
	createdJob, err := createJob(ctx, q, job)
	if err != nil {
		return websites.WebsiteDomain{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return websites.WebsiteDomain{}, jobs.Job{}, err
	}
	return websiteDomainValue(row), createdJob, nil
}

func (s *Store) QueueWebsiteDomainRename(ctx context.Context, id, hostname string, job jobs.Job) (websites.WebsiteDomain, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return websites.WebsiteDomain{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	if err := ensureHostnameAvailable(ctx, q, hostname); err != nil {
		return websites.WebsiteDomain{}, jobs.Job{}, err
	}
	row, err := q.QueueWebsiteDomainRename(ctx, dbgen.QueueWebsiteDomainRenameParams{Hostname: hostname, UpdatedAt: timeValue(time.Now().UTC()), ID: id})
	if errors.Is(err, sql.ErrNoRows) {
		return websites.WebsiteDomain{}, jobs.Job{}, websites.ErrWebsiteDomainBusy
	}
	if err != nil {
		return websites.WebsiteDomain{}, jobs.Job{}, err
	}
	createdJob, err := createJob(ctx, q, job)
	if err != nil {
		return websites.WebsiteDomain{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return websites.WebsiteDomain{}, jobs.Job{}, err
	}
	return websiteDomainValue(row), createdJob, nil
}

func (s *Store) UpdateWebsiteStatus(ctx context.Context, id, status string) error {
	return s.queries.UpdateWebsiteStatus(ctx, dbgen.UpdateWebsiteStatusParams{Status: status, UpdatedAt: timeValue(time.Now().UTC()), ID: id})
}

func (s *Store) UpdateWebsiteDomainStatus(ctx context.Context, id, status string) error {
	return s.queries.UpdateWebsiteDomainStatus(ctx, dbgen.UpdateWebsiteDomainStatusParams{Status: status, UpdatedAt: timeValue(time.Now().UTC()), ID: id})
}

func (s *Store) CompleteWebsiteDomainRename(ctx context.Context, id string) error {
	return s.queries.CompleteWebsiteDomainRename(ctx, dbgen.CompleteWebsiteDomainRenameParams{UpdatedAt: timeValue(time.Now().UTC()), ID: id})
}

func (s *Store) FailWebsiteDomainRename(ctx context.Context, id string) error {
	return s.queries.FailWebsiteDomainRename(ctx, dbgen.FailWebsiteDomainRenameParams{UpdatedAt: timeValue(time.Now().UTC()), ID: id})
}

func (s *Store) DeleteWebsite(ctx context.Context, id string) error {
	return s.queries.DeleteWebsite(ctx, id)
}

func (s *Store) DeleteWebsiteDomain(ctx context.Context, id string) error {
	return s.queries.DeleteWebsiteDomain(ctx, id)
}

func ensureHostnameAvailable(ctx context.Context, q *dbgen.Queries, hostname string) error {
	exists, err := q.WebsiteHostnameExists(ctx, hostname)
	if err != nil {
		return err
	}
	if exists {
		return websites.ErrHostnameExists
	}
	return nil
}

func websiteValues(rows []dbgen.Website) []websites.Website {
	values := make([]websites.Website, 0, len(rows))
	for _, row := range rows {
		values = append(values, websiteValue(row))
	}
	return values
}

func websiteValue(row dbgen.Website) websites.Website {
	return websites.Website{ID: row.ID, AccountID: row.AccountID, NodeID: row.NodeID, Name: row.Name, Kind: row.Kind, DocumentRoot: row.DocumentRoot, WebDriver: row.WebDriver, RuntimeDriver: row.RuntimeDriver, RuntimeVersion: row.RuntimeVersion, Status: row.Status, Enabled: row.Enabled != 0, CreatedAt: timeFrom(row.CreatedAt), UpdatedAt: timeFrom(row.UpdatedAt)}
}

func websiteDomainValues(rows []dbgen.WebsiteDomain) []websites.WebsiteDomain {
	values := make([]websites.WebsiteDomain, 0, len(rows))
	for _, row := range rows {
		values = append(values, websiteDomainValue(row))
	}
	return values
}

func websiteDomainValue(row dbgen.WebsiteDomain) websites.WebsiteDomain {
	return websites.WebsiteDomain{ID: row.ID, WebsiteID: row.WebsiteID, Hostname: row.Hostname, Kind: row.Kind, Status: row.Status, Enabled: row.Enabled != 0, PreviousHostname: row.PreviousHostname.String, CreatedAt: timeFrom(row.CreatedAt), UpdatedAt: timeFrom(row.UpdatedAt)}
}
