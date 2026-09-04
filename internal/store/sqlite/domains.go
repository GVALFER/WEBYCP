package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/GVALFER/WEBYCP/internal/domains"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite/dbgen"
)

func (s *Store) CreateDomainProvision(
	ctx context.Context,
	domain domains.Domain,
	job jobs.Job,
) (domains.Domain, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domains.Domain{}, jobs.Job{}, fmt.Errorf("begin domain creation: %w", err)
	}
	defer tx.Rollback()
	queries := s.queries.WithTx(tx)

	if err := ensureDomainNameAvailable(ctx, queries, domain.Name); err != nil {
		return domains.Domain{}, jobs.Job{}, err
	}
	created, err := queries.CreateDomain(ctx, dbgen.CreateDomainParams{
		ID: domain.ID, AccountID: domain.AccountID, NodeID: domain.NodeID,
		Name: domain.Name, PhpVersion: domain.PHPVersion,
		CreatedAt: timeValue(domain.CreatedAt), UpdatedAt: timeValue(domain.UpdatedAt),
	})
	if err != nil {
		return domains.Domain{}, jobs.Job{}, fmt.Errorf("insert domain: %w", err)
	}
	createdJob, err := queries.CreateJob(ctx, dbgen.CreateJobParams{
		ID: job.ID, NodeID: nullString(job.NodeID), UserID: nullString(job.UserID),
		Kind: job.Kind, Payload: job.Payload, MaxAttempts: job.MaxAttempts,
		CreatedAt: timeValue(job.CreatedAt),
	})
	if err != nil {
		return domains.Domain{}, jobs.Job{}, fmt.Errorf("insert domain job: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domains.Domain{}, jobs.Job{}, fmt.Errorf("commit domain creation: %w", err)
	}
	return domainValue(created), jobValue(createdJob), nil
}

func (s *Store) CreateAliasProvision(
	ctx context.Context,
	alias domains.Alias,
	job jobs.Job,
) (domains.Alias, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domains.Alias{}, jobs.Job{}, fmt.Errorf("begin alias creation: %w", err)
	}
	defer tx.Rollback()
	queries := s.queries.WithTx(tx)
	if err := ensureDomainNameAvailable(ctx, queries, alias.Name); err != nil {
		return domains.Alias{}, jobs.Job{}, err
	}
	created, err := queries.CreateDomainAlias(ctx, dbgen.CreateDomainAliasParams{
		ID: alias.ID, DomainID: alias.DomainID, Name: alias.Name,
		CreatedAt: timeValue(alias.CreatedAt), UpdatedAt: timeValue(alias.UpdatedAt),
	})
	if err != nil {
		return domains.Alias{}, jobs.Job{}, fmt.Errorf("insert domain alias: %w", err)
	}
	createdJob, err := queries.CreateJob(ctx, dbgen.CreateJobParams{
		ID: job.ID, NodeID: nullString(job.NodeID), UserID: nullString(job.UserID),
		Kind: job.Kind, Payload: job.Payload, MaxAttempts: job.MaxAttempts,
		CreatedAt: timeValue(job.CreatedAt),
	})
	if err != nil {
		return domains.Alias{}, jobs.Job{}, fmt.Errorf("insert alias job: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domains.Alias{}, jobs.Job{}, fmt.Errorf("commit alias creation: %w", err)
	}
	return aliasValue(created), jobValue(createdJob), nil
}

func (s *Store) Domain(ctx context.Context, id string) (domains.Domain, error) {
	row, err := s.queries.GetDomain(ctx, id)
	if err != nil {
		return domains.Domain{}, err
	}
	return domainValue(row), nil
}

func (s *Store) Domains(ctx context.Context, userID string, admin bool) ([]domains.Domain, error) {
	var (
		rows []dbgen.Domain
		err  error
	)
	if admin {
		rows, err = s.queries.ListDomains(ctx)
	} else {
		rows, err = s.queries.ListUserDomains(ctx, userID)
	}
	if err != nil {
		return nil, err
	}
	result := make([]domains.Domain, 0, len(rows))
	for _, row := range rows {
		result = append(result, domainValue(row))
	}
	return result, nil
}

func (s *Store) DomainPage(
	ctx context.Context, userID string, admin bool, query pagination.Query,
) (pagination.Result[domains.Domain], error) {
	var (
		total int64
		rows  []dbgen.Domain
		err   error
	)
	if admin {
		total, err = s.queries.CountDomains(ctx)
	} else {
		total, err = s.queries.CountUserDomains(ctx, userID)
	}
	if err != nil {
		return pagination.Result[domains.Domain]{}, err
	}
	query = pagination.Clamp(query, total)
	if admin {
		rows, err = s.queries.ListDomainsPage(ctx, dbgen.ListDomainsPageParams{
			PageOffset: pagination.Offset(query), PageSize: int64(query.Size),
		})
	} else {
		rows, err = s.queries.ListUserDomainsPage(ctx, dbgen.ListUserDomainsPageParams{
			UserID: userID, PageOffset: pagination.Offset(query), PageSize: int64(query.Size),
		})
	}
	if err != nil {
		return pagination.Result[domains.Domain]{}, err
	}
	items := make([]domains.Domain, 0, len(rows))
	for _, row := range rows {
		items = append(items, domainValue(row))
	}
	return pagination.Result[domains.Domain]{Items: items, Query: query, Total: total}, nil
}

func (s *Store) Alias(ctx context.Context, id string) (domains.Alias, error) {
	row, err := s.queries.GetDomainAlias(ctx, id)
	if err != nil {
		return domains.Alias{}, err
	}
	return aliasValue(row), nil
}

func (s *Store) Aliases(ctx context.Context, domainID string) ([]domains.Alias, error) {
	rows, err := s.queries.ListDomainAliases(ctx, domainID)
	if err != nil {
		return nil, err
	}
	return aliasValues(rows), nil
}

func (s *Store) AliasPage(
	ctx context.Context, domainID string, query pagination.Query,
) (pagination.Result[domains.Alias], error) {
	total, err := s.queries.CountDomainAliases(ctx, domainID)
	if err != nil {
		return pagination.Result[domains.Alias]{}, err
	}
	query = pagination.Clamp(query, total)
	rows, err := s.queries.ListDomainAliasesPage(ctx, dbgen.ListDomainAliasesPageParams{
		DomainID: domainID, PageOffset: pagination.Offset(query), PageSize: int64(query.Size),
	})
	if err != nil {
		return pagination.Result[domains.Alias]{}, err
	}
	return pagination.Result[domains.Alias]{
		Items: aliasValues(rows), Query: query, Total: total,
	}, nil
}

func (s *Store) EnabledAliases(ctx context.Context, domainID string) ([]domains.Alias, error) {
	rows, err := s.queries.ListEnabledDomainAliases(ctx, domainID)
	if err != nil {
		return nil, err
	}
	return aliasValues(rows), nil
}

func (s *Store) QueueDomainAction(
	ctx context.Context,
	id string,
	enabled bool,
	job jobs.Job,
) (domains.Domain, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domains.Domain{}, jobs.Job{}, fmt.Errorf("begin domain action: %w", err)
	}
	defer tx.Rollback()
	queries := s.queries.WithTx(tx)
	updated, err := queries.QueueDomainAction(ctx, dbgen.QueueDomainActionParams{
		Enabled: boolValue(enabled), UpdatedAt: timeValue(time.Now().UTC()), ID: id,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domains.Domain{}, jobs.Job{}, domains.ErrBusy
		}
		return domains.Domain{}, jobs.Job{}, fmt.Errorf("queue domain action: %w", err)
	}
	createdJob, err := queries.CreateJob(ctx, dbgen.CreateJobParams{
		ID: job.ID, NodeID: nullString(job.NodeID), UserID: nullString(job.UserID),
		Kind: job.Kind, Payload: job.Payload, MaxAttempts: job.MaxAttempts,
		CreatedAt: timeValue(job.CreatedAt),
	})
	if err != nil {
		return domains.Domain{}, jobs.Job{}, fmt.Errorf("insert domain action job: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domains.Domain{}, jobs.Job{}, fmt.Errorf("commit domain action: %w", err)
	}
	return domainValue(updated), jobValue(createdJob), nil
}

func (s *Store) QueueAliasAction(
	ctx context.Context,
	id string,
	enabled bool,
	job jobs.Job,
) (domains.Alias, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domains.Alias{}, jobs.Job{}, fmt.Errorf("begin alias action: %w", err)
	}
	defer tx.Rollback()
	queries := s.queries.WithTx(tx)
	updated, err := queries.QueueDomainAliasAction(ctx, dbgen.QueueDomainAliasActionParams{
		Enabled: boolValue(enabled), UpdatedAt: timeValue(time.Now().UTC()), ID: id,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domains.Alias{}, jobs.Job{}, domains.ErrBusy
		}
		return domains.Alias{}, jobs.Job{}, fmt.Errorf("queue alias action: %w", err)
	}
	createdJob, err := queries.CreateJob(ctx, dbgen.CreateJobParams{
		ID: job.ID, NodeID: nullString(job.NodeID), UserID: nullString(job.UserID),
		Kind: job.Kind, Payload: job.Payload, MaxAttempts: job.MaxAttempts,
		CreatedAt: timeValue(job.CreatedAt),
	})
	if err != nil {
		return domains.Alias{}, jobs.Job{}, fmt.Errorf("insert alias action job: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domains.Alias{}, jobs.Job{}, fmt.Errorf("commit alias action: %w", err)
	}
	return aliasValue(updated), jobValue(createdJob), nil
}

func (s *Store) QueueDomainRename(
	ctx context.Context,
	id, name string,
	job jobs.Job,
) (domains.Domain, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domains.Domain{}, jobs.Job{}, fmt.Errorf("begin domain rename: %w", err)
	}
	defer tx.Rollback()
	queries := s.queries.WithTx(tx)
	if err := ensureDomainNameAvailable(ctx, queries, name); err != nil {
		return domains.Domain{}, jobs.Job{}, err
	}
	updated, err := queries.QueueDomainRename(ctx, dbgen.QueueDomainRenameParams{
		Name: name, UpdatedAt: timeValue(time.Now().UTC()), ID: id,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domains.Domain{}, jobs.Job{}, domains.ErrBusy
		}
		return domains.Domain{}, jobs.Job{}, fmt.Errorf("queue domain rename: %w", err)
	}
	createdJob, err := queries.CreateJob(ctx, dbgen.CreateJobParams{
		ID: job.ID, NodeID: nullString(job.NodeID), UserID: nullString(job.UserID),
		Kind: job.Kind, Payload: job.Payload, MaxAttempts: job.MaxAttempts,
		CreatedAt: timeValue(job.CreatedAt),
	})
	if err != nil {
		return domains.Domain{}, jobs.Job{}, fmt.Errorf("insert domain rename job: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domains.Domain{}, jobs.Job{}, fmt.Errorf("commit domain rename: %w", err)
	}
	return domainValue(updated), jobValue(createdJob), nil
}

func (s *Store) QueueAliasRename(
	ctx context.Context,
	id, name string,
	job jobs.Job,
) (domains.Alias, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domains.Alias{}, jobs.Job{}, fmt.Errorf("begin alias rename: %w", err)
	}
	defer tx.Rollback()
	queries := s.queries.WithTx(tx)
	if err := ensureDomainNameAvailable(ctx, queries, name); err != nil {
		return domains.Alias{}, jobs.Job{}, err
	}
	updated, err := queries.QueueDomainAliasRename(ctx, dbgen.QueueDomainAliasRenameParams{
		Name: name, UpdatedAt: timeValue(time.Now().UTC()), ID: id,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domains.Alias{}, jobs.Job{}, domains.ErrBusy
		}
		return domains.Alias{}, jobs.Job{}, fmt.Errorf("queue alias rename: %w", err)
	}
	createdJob, err := queries.CreateJob(ctx, dbgen.CreateJobParams{
		ID: job.ID, NodeID: nullString(job.NodeID), UserID: nullString(job.UserID),
		Kind: job.Kind, Payload: job.Payload, MaxAttempts: job.MaxAttempts,
		CreatedAt: timeValue(job.CreatedAt),
	})
	if err != nil {
		return domains.Alias{}, jobs.Job{}, fmt.Errorf("insert alias rename job: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domains.Alias{}, jobs.Job{}, fmt.Errorf("commit alias rename: %w", err)
	}
	return aliasValue(updated), jobValue(createdJob), nil
}

func (s *Store) UpdateDomainStatus(ctx context.Context, id, status string) error {
	return s.queries.UpdateDomainStatus(ctx, dbgen.UpdateDomainStatusParams{
		Status: status, UpdatedAt: timeValue(time.Now().UTC()), ID: id,
	})
}

func (s *Store) UpdateAliasStatus(ctx context.Context, id, status string) error {
	return s.queries.UpdateDomainAliasStatus(ctx, dbgen.UpdateDomainAliasStatusParams{
		Status: status, UpdatedAt: timeValue(time.Now().UTC()), ID: id,
	})
}

func (s *Store) DeleteDomain(ctx context.Context, id string) error {
	return s.queries.DeleteDomain(ctx, id)
}

func (s *Store) DeleteAlias(ctx context.Context, id string) error {
	return s.queries.DeleteDomainAlias(ctx, id)
}

func (s *Store) CompleteDomainRename(ctx context.Context, id string) error {
	return s.queries.CompleteDomainRename(ctx, dbgen.CompleteDomainRenameParams{
		UpdatedAt: timeValue(time.Now().UTC()), ID: id,
	})
}

func (s *Store) CompleteAliasRename(ctx context.Context, id string) error {
	return s.queries.CompleteDomainAliasRename(ctx, dbgen.CompleteDomainAliasRenameParams{
		UpdatedAt: timeValue(time.Now().UTC()), ID: id,
	})
}

func (s *Store) FailDomainRename(ctx context.Context, id string) error {
	return s.queries.FailDomainRename(ctx, dbgen.FailDomainRenameParams{
		UpdatedAt: timeValue(time.Now().UTC()), ID: id,
	})
}

func (s *Store) FailAliasRename(ctx context.Context, id string) error {
	return s.queries.FailDomainAliasRename(ctx, dbgen.FailDomainAliasRenameParams{
		UpdatedAt: timeValue(time.Now().UTC()), ID: id,
	})
}

func domainValue(row dbgen.Domain) domains.Domain {
	return domains.Domain{
		ID: row.ID, AccountID: row.AccountID, NodeID: row.NodeID, Name: row.Name,
		Status: row.Status, PHPVersion: row.PhpVersion, Enabled: row.Enabled == 1,
		PreviousName: row.PreviousName.String,
		CreatedAt:    timeFrom(row.CreatedAt), UpdatedAt: timeFrom(row.UpdatedAt),
	}
}

func ensureDomainNameAvailable(ctx context.Context, queries *dbgen.Queries, name string) error {
	exists, err := queries.DomainNameExists(ctx, name)
	if err != nil {
		return fmt.Errorf("check domain name: %w", err)
	}
	if exists {
		return domains.ErrNameExists
	}
	return nil
}

func aliasValues(rows []dbgen.DomainAlias) []domains.Alias {
	result := make([]domains.Alias, 0, len(rows))
	for _, row := range rows {
		result = append(result, aliasValue(row))
	}
	return result
}

func aliasValue(row dbgen.DomainAlias) domains.Alias {
	return domains.Alias{
		ID: row.ID, DomainID: row.DomainID, Name: row.Name,
		Status: row.Status, Enabled: row.Enabled == 1,
		PreviousName: row.PreviousName.String,
		CreatedAt:    timeFrom(row.CreatedAt), UpdatedAt: timeFrom(row.UpdatedAt),
	}
}

func boolValue(value bool) int64 {
	if value {
		return 1
	}
	return 0
}
