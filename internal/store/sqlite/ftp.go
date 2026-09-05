package sqlite

import (
	"context"
	"time"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/ftp"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite/dbgen"
)

func (s *Store) CreateFTP(ctx context.Context, value ftp.Credential, job jobs.Job) (ftp.Account, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ftp.Account{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	if err := requireFTP(ctx, q, value.Account, true); err != nil {
		return ftp.Account{}, jobs.Job{}, err
	}
	if err := requireCapacity(ctx, tx, value.AccountID, limitFTPAccounts); err != nil {
		return ftp.Account{}, jobs.Job{}, err
	}
	if err := q.CreateFTP(ctx, dbgen.CreateFTPParams{
		ID: value.ID, AccountID: value.AccountID, NodeID: value.NodeID,
		Username: value.Username, PasswordHash: value.PasswordHash, Enabled: boolValue(value.Enabled),
		CreatedAt: timeValue(value.CreatedAt), UpdatedAt: timeValue(value.UpdatedAt),
	}); err != nil {
		return ftp.Account{}, jobs.Job{}, err
	}
	created, err := createJob(ctx, q, job)
	if err != nil {
		return ftp.Account{}, jobs.Job{}, err
	}
	row, err := q.GetFTP(ctx, value.ID)
	if err != nil {
		return ftp.Account{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return ftp.Account{}, jobs.Job{}, err
	}
	return ftpValue(row).Account, created, nil
}

func (s *Store) ChangeFTP(ctx context.Context, id string, change ftp.Changes, job jobs.Job) (ftp.Account, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ftp.Account{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	row, err := q.GetFTP(ctx, id)
	if err != nil {
		return ftp.Account{}, jobs.Job{}, err
	}
	value := ftpValue(row)
	if value.Deleting && !change.Deleting {
		return ftp.Account{}, jobs.Job{}, ftp.ErrDeleting
	}
	if change.Username != nil {
		value.Username = *change.Username
	}
	if change.PasswordHash != nil {
		value.PasswordHash = *change.PasswordHash
	}
	if change.Enabled != nil {
		value.Enabled = *change.Enabled
	}
	value.Deleting = change.Deleting
	if err := requireFTP(ctx, q, value.Account, false); err != nil {
		return ftp.Account{}, jobs.Job{}, err
	}
	value.Status, value.UpdatedAt = "pending", time.Now().UTC()
	if err := q.UpdateFTP(ctx, dbgen.UpdateFTPParams{
		ID: id, Username: value.Username, PasswordHash: value.PasswordHash,
		Enabled: boolValue(value.Enabled), Deleting: boolValue(value.Deleting), UpdatedAt: timeValue(value.UpdatedAt),
	}); err != nil {
		return ftp.Account{}, jobs.Job{}, err
	}
	created, err := createJob(ctx, q, job)
	if err != nil {
		return ftp.Account{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return ftp.Account{}, jobs.Job{}, err
	}
	return value.Account, created, nil
}

// Serialize FTP writes per Account until its durable Job has finished. Merge
// PATCH fields inside this same transaction, not from a stale service read.
func requireFTP(ctx context.Context, q *dbgen.Queries, value ftp.Account, creating bool) error {
	account, err := q.GetAccount(ctx, value.AccountID)
	if err != nil {
		return err
	}
	if (account.Status != "active" && account.Status != "disabled") ||
		(creating && (account.Status != "active" || account.Enabled == 0)) {
		return accounts.ErrBusy
	}
	busy, err := q.FTPBusy(ctx, value.AccountID)
	if err != nil {
		return err
	}
	if busy {
		return ftp.ErrBusy
	}
	exists, err := q.FTPNameExists(ctx, dbgen.FTPNameExistsParams{NodeID: value.NodeID, Username: value.Username, ID: value.ID})
	if err != nil {
		return err
	}
	if exists {
		return ftp.ErrNameExists
	}
	return nil
}

func (s *Store) FTP(ctx context.Context, id string) (ftp.Credential, error) {
	row, err := s.queries.GetFTP(ctx, id)
	return ftpValue(row), err
}

func (s *Store) FTPPage(ctx context.Context, userID string, admin bool, query pagination.Query) (pagination.Result[ftp.Account], error) {
	total, err := s.queries.CountFTP(ctx, dbgen.CountFTPParams{IsAdmin: admin, UserID: userID})
	if err != nil {
		return pagination.Result[ftp.Account]{}, err
	}
	query = pagination.Clamp(query, total)
	rows, err := s.queries.ListFTPPage(ctx, dbgen.ListFTPPageParams{
		IsAdmin: admin, UserID: userID, PageSize: int64(query.Size), PageOffset: pagination.Offset(query),
	})
	if err != nil {
		return pagination.Result[ftp.Account]{}, err
	}
	items := make([]ftp.Account, 0, len(rows))
	for _, row := range rows {
		items = append(items, ftpValue(row).Account)
	}
	return pagination.Result[ftp.Account]{Items: items, Query: query, Total: total}, nil
}

func (s *Store) AccountFTP(ctx context.Context, id string) ([]ftp.Credential, error) {
	rows, err := s.queries.AccountFTP(ctx, id)
	if err != nil {
		return nil, err
	}
	items := make([]ftp.Credential, 0, len(rows))
	for _, row := range rows {
		items = append(items, ftpValue(row))
	}
	return items, nil
}

func (s *Store) FinishFTP(ctx context.Context, accountID string, failed bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	if !failed {
		if err := q.DeleteFTP(ctx, accountID); err != nil {
			return err
		}
	}
	if err := q.SetFTPStatuses(ctx, dbgen.SetFTPStatusesParams{AccountID: accountID,
		Failed: boolValue(failed), UpdatedAt: timeValue(time.Now().UTC()),
	}); err != nil {
		return err
	}
	return tx.Commit()
}

func ftpValue(row dbgen.FtpOverview) ftp.Credential {
	return ftp.Credential{
		Account: ftp.Account{
			ID: row.ID, AccountID: row.AccountID, NodeID: row.NodeID, Username: row.Username,
			AccountName: row.AccountName, AccountStatus: row.AccountStatus, SystemUser: row.SystemUser,
			Enabled: row.Enabled != 0, Deleting: row.Deleting != 0, Status: row.Status,
			CreatedAt: timeFrom(row.CreatedAt), UpdatedAt: timeFrom(row.UpdatedAt),
		},
		PasswordHash: row.PasswordHash,
	}
}
