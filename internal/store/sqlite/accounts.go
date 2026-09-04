package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite/dbgen"
)

func (s *Store) CreateProvision(
	ctx context.Context,
	account accounts.Account,
	ownerID string,
	job jobs.Job,
) (accounts.Account, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return accounts.Account{}, jobs.Job{}, fmt.Errorf("begin account creation: %w", err)
	}
	defer tx.Rollback()
	queries := s.queries.WithTx(tx)

	exists, err := queries.AccountNameExists(ctx, account.Name)
	if err != nil {
		return accounts.Account{}, jobs.Job{}, fmt.Errorf("check account name: %w", err)
	}
	if exists {
		return accounts.Account{}, jobs.Job{}, accounts.ErrNameExists
	}
	created, err := queries.CreateAccount(ctx, dbgen.CreateAccountParams{
		ID: account.ID, NodeID: nullString(account.NodeID), Name: account.Name,
		SystemUser: account.SystemUser, CreatedAt: timeValue(account.CreatedAt),
		UpdatedAt: timeValue(account.UpdatedAt),
	})
	if err != nil {
		return accounts.Account{}, jobs.Job{}, fmt.Errorf("insert account: %w", err)
	}
	if err := queries.AddAccountMember(ctx, dbgen.AddAccountMemberParams{
		AccountID: account.ID, UserID: ownerID, CreatedAt: timeValue(account.CreatedAt),
	}); err != nil {
		return accounts.Account{}, jobs.Job{}, fmt.Errorf("add account owner: %w", err)
	}
	createdJob, err := queries.CreateJob(ctx, dbgen.CreateJobParams{
		ID: job.ID, NodeID: nullString(job.NodeID), UserID: nullString(job.UserID),
		Kind: job.Kind, Payload: job.Payload, MaxAttempts: job.MaxAttempts,
		CreatedAt: timeValue(job.CreatedAt),
	})
	if err != nil {
		return accounts.Account{}, jobs.Job{}, fmt.Errorf("insert account job: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return accounts.Account{}, jobs.Job{}, fmt.Errorf("commit account creation: %w", err)
	}

	return accountValue(created), jobValue(createdJob), nil
}

func (s *Store) Account(ctx context.Context, id string) (accounts.Account, error) {
	row, err := s.queries.GetAccount(ctx, id)
	if err != nil {
		return accounts.Account{}, err
	}
	return accountValue(row), nil
}

func (s *Store) AccountMember(ctx context.Context, accountID, userID string) (bool, error) {
	return s.queries.AccountMemberExists(ctx, dbgen.AccountMemberExistsParams{
		AccountID: accountID, UserID: userID,
	})
}

func (s *Store) Accounts(ctx context.Context, userID string, admin bool) ([]accounts.Account, error) {
	var (
		rows []dbgen.Account
		err  error
	)
	if admin {
		rows, err = s.queries.ListAccounts(ctx)
	} else {
		rows, err = s.queries.ListUserAccounts(ctx, userID)
	}
	if err != nil {
		return nil, err
	}
	result := make([]accounts.Account, 0, len(rows))
	for _, row := range rows {
		result = append(result, accountValue(row))
	}
	return result, nil
}

func (s *Store) AccountPage(
	ctx context.Context, userID string, admin bool, query pagination.Query,
) (pagination.Result[accounts.Account], error) {
	var (
		rows  []dbgen.Account
		total int64
		err   error
	)
	if admin {
		total, err = s.queries.CountAccounts(ctx)
	} else {
		total, err = s.queries.CountUserAccounts(ctx, userID)
	}
	if err != nil {
		return pagination.Result[accounts.Account]{}, err
	}

	query = pagination.Clamp(query, total)
	if admin {
		rows, err = s.queries.ListAccountsPage(ctx, dbgen.ListAccountsPageParams{
			PageOffset: pagination.Offset(query), PageSize: int64(query.Size),
		})
	} else {
		rows, err = s.queries.ListUserAccountsPage(ctx, dbgen.ListUserAccountsPageParams{
			UserID: userID, PageOffset: pagination.Offset(query), PageSize: int64(query.Size),
		})
	}
	if err != nil {
		return pagination.Result[accounts.Account]{}, err
	}

	items := make([]accounts.Account, 0, len(rows))
	for _, row := range rows {
		items = append(items, accountValue(row))
	}
	return pagination.Result[accounts.Account]{Items: items, Query: query, Total: total}, nil
}

func (s *Store) QueueAction(
	ctx context.Context, id string, enabled bool, job jobs.Job,
) (accounts.Account, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return accounts.Account{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	queries := s.queries.WithTx(tx)
	row, err := queries.QueueAccountAction(ctx, dbgen.QueueAccountActionParams{
		Enabled: boolValue(enabled), UpdatedAt: timeValue(time.Now().UTC()), ID: id,
	})
	if err != nil {
		return accounts.Account{}, jobs.Job{}, err
	}
	created, err := queries.CreateJob(ctx, dbgen.CreateJobParams{
		ID: job.ID, NodeID: nullString(job.NodeID), UserID: nullString(job.UserID),
		Kind: job.Kind, Payload: job.Payload, MaxAttempts: job.MaxAttempts,
		CreatedAt: timeValue(job.CreatedAt),
	})
	if err != nil {
		return accounts.Account{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return accounts.Account{}, jobs.Job{}, err
	}
	return accountValue(row), jobValue(created), nil
}

func (s *Store) ResourceCount(ctx context.Context, id string) (int64, error) {
	return s.queries.AccountResourceCount(ctx, id)
}

func (s *Store) Delete(ctx context.Context, id string) error {
	return s.queries.DeleteAccount(ctx, id)
}

func (s *Store) UpdateStatus(ctx context.Context, id, status string) error {
	return s.queries.UpdateAccountStatus(ctx, dbgen.UpdateAccountStatusParams{
		Status: status, UpdatedAt: timeValue(time.Now().UTC()), ID: id,
	})
}

func accountValue(row dbgen.Account) accounts.Account {
	return accounts.Account{
		ID: row.ID, NodeID: row.NodeID.String, Name: row.Name, SystemUser: row.SystemUser,
		Status: row.Status, Enabled: row.Enabled != 0,
		CreatedAt: timeFrom(row.CreatedAt), UpdatedAt: timeFrom(row.UpdatedAt),
	}
}
