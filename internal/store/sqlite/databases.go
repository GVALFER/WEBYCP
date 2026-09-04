package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/GVALFER/WEBYCP/internal/databases"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite/dbgen"
)

func (s *Store) CreateDatabase(
	ctx context.Context, value databases.Database, job jobs.Job,
) (databases.Database, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return databases.Database{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	if err := requireCapacity(ctx, tx, value.AccountID, limitDatabases); err != nil {
		return databases.Database{}, jobs.Job{}, err
	}
	exists, err := q.DatabaseNameExists(ctx, dbgen.DatabaseNameExistsParams{AccountID: value.AccountID, Name: value.Name})
	if err != nil {
		return databases.Database{}, jobs.Job{}, err
	}
	if exists {
		return databases.Database{}, jobs.Job{}, databases.ErrNameExists
	}
	row, err := q.CreateDatabase(ctx, dbgen.CreateDatabaseParams{
		ID: value.ID, AccountID: value.AccountID, NodeID: value.NodeID, Name: value.Name,
		SystemName: value.SystemName, CreatedAt: timeValue(value.CreatedAt), UpdatedAt: timeValue(value.UpdatedAt),
	})
	if err != nil {
		return databases.Database{}, jobs.Job{}, err
	}
	created, err := createJob(ctx, q, job)
	if err != nil {
		return databases.Database{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return databases.Database{}, jobs.Job{}, err
	}
	return databaseValue(row), created, nil
}

func (s *Store) Databases(ctx context.Context, userID string, admin bool) ([]databases.Database, error) {
	rows, err := s.queries.ListDatabases(ctx, dbgen.ListDatabasesParams{Column1: admin, UserID: userID})
	if err != nil {
		return nil, err
	}
	result := make([]databases.Database, 0, len(rows))
	for _, row := range rows {
		result = append(result, databaseValue(row))
	}
	return result, nil
}

func (s *Store) DatabasePage(
	ctx context.Context, userID string, admin bool, query pagination.Query,
) (pagination.Result[databases.Database], error) {
	total, err := s.queries.CountDatabases(ctx, dbgen.CountDatabasesParams{
		IsAdmin: admin, UserID: userID,
	})
	if err != nil {
		return pagination.Result[databases.Database]{}, err
	}
	query = pagination.Clamp(query, total)
	rows, err := s.queries.ListDatabasesPage(ctx, dbgen.ListDatabasesPageParams{
		IsAdmin: admin, UserID: userID,
		PageOffset: pagination.Offset(query), PageSize: int64(query.Size),
	})
	if err != nil {
		return pagination.Result[databases.Database]{}, err
	}
	items := make([]databases.Database, 0, len(rows))
	for _, row := range rows {
		items = append(items, databaseValue(row))
	}
	return pagination.Result[databases.Database]{Items: items, Query: query, Total: total}, nil
}

func (s *Store) Database(ctx context.Context, id string) (databases.Database, error) {
	row, err := s.queries.GetDatabase(ctx, id)
	return databaseValue(row), err
}

func (s *Store) QueueDatabaseDelete(
	ctx context.Context, id string, job jobs.Job,
) (databases.Database, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return databases.Database{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	row, err := q.QueueDatabaseDelete(ctx, dbgen.QueueDatabaseDeleteParams{UpdatedAt: timeValue(time.Now().UTC()), ID: id})
	if err != nil {
		return databases.Database{}, jobs.Job{}, err
	}
	created, err := createJob(ctx, q, job)
	if err != nil {
		return databases.Database{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return databases.Database{}, jobs.Job{}, err
	}
	return databaseValue(row), created, nil
}

func (s *Store) SetDatabaseStatus(ctx context.Context, id, status string) error {
	return s.queries.UpdateDatabaseStatus(ctx, dbgen.UpdateDatabaseStatusParams{Status: status, UpdatedAt: timeValue(time.Now().UTC()), ID: id})
}

func (s *Store) DeleteDatabase(ctx context.Context, id string) error {
	return s.queries.DeleteDatabase(ctx, id)
}

func (s *Store) CreateUser(
	ctx context.Context, value databases.User, job jobs.Job,
) (databases.User, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return databases.User{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	if err := requireCapacity(ctx, tx, value.AccountID, limitDatabaseUsers); err != nil {
		return databases.User{}, jobs.Job{}, err
	}
	exists, err := q.DatabaseUserNameExists(ctx, dbgen.DatabaseUserNameExistsParams{AccountID: value.AccountID, Name: value.Name})
	if err != nil {
		return databases.User{}, jobs.Job{}, err
	}
	if exists {
		return databases.User{}, jobs.Job{}, databases.ErrNameExists
	}
	row, err := q.CreateDatabaseUser(ctx, dbgen.CreateDatabaseUserParams{
		ID: value.ID, AccountID: value.AccountID, NodeID: value.NodeID, Name: value.Name,
		SystemName: value.SystemName, CreatedAt: timeValue(value.CreatedAt), UpdatedAt: timeValue(value.UpdatedAt),
	})
	if err != nil {
		return databases.User{}, jobs.Job{}, err
	}
	created, err := createJob(ctx, q, job)
	if err != nil {
		return databases.User{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return databases.User{}, jobs.Job{}, err
	}
	return databaseUserValue(row), created, nil
}

func (s *Store) Users(ctx context.Context, userID string, admin bool) ([]databases.User, error) {
	rows, err := s.queries.ListDatabaseUsers(ctx, dbgen.ListDatabaseUsersParams{Column1: admin, UserID: userID})
	if err != nil {
		return nil, err
	}
	result := make([]databases.User, 0, len(rows))
	for _, row := range rows {
		result = append(result, databaseUserValue(row))
	}
	return result, nil
}

func (s *Store) UserPage(
	ctx context.Context, userID string, admin bool, query pagination.Query,
) (pagination.Result[databases.User], error) {
	total, err := s.queries.CountDatabaseUsers(ctx, dbgen.CountDatabaseUsersParams{
		IsAdmin: admin, UserID: userID,
	})
	if err != nil {
		return pagination.Result[databases.User]{}, err
	}
	query = pagination.Clamp(query, total)
	rows, err := s.queries.ListDatabaseUsersPage(ctx, dbgen.ListDatabaseUsersPageParams{
		IsAdmin: admin, UserID: userID,
		PageOffset: pagination.Offset(query), PageSize: int64(query.Size),
	})
	if err != nil {
		return pagination.Result[databases.User]{}, err
	}
	items := make([]databases.User, 0, len(rows))
	for _, row := range rows {
		items = append(items, databaseUserValue(row))
	}
	return pagination.Result[databases.User]{Items: items, Query: query, Total: total}, nil
}

func (s *Store) User(ctx context.Context, id string) (databases.User, error) {
	row, err := s.queries.GetDatabaseUser(ctx, id)
	return databaseUserValue(row), err
}

func (s *Store) QueueUserDelete(
	ctx context.Context, id string, job jobs.Job,
) (databases.User, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return databases.User{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	row, err := q.QueueDatabaseUserDelete(ctx, dbgen.QueueDatabaseUserDeleteParams{UpdatedAt: timeValue(time.Now().UTC()), ID: id})
	if err != nil {
		return databases.User{}, jobs.Job{}, err
	}
	created, err := createJob(ctx, q, job)
	if err != nil {
		return databases.User{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return databases.User{}, jobs.Job{}, err
	}
	return databaseUserValue(row), created, nil
}

func (s *Store) SetUserStatus(ctx context.Context, id, status string) error {
	return s.queries.UpdateDatabaseUserStatus(ctx, dbgen.UpdateDatabaseUserStatusParams{Status: status, UpdatedAt: timeValue(time.Now().UTC()), ID: id})
}

func (s *Store) DeleteUser(ctx context.Context, id string) error {
	return s.queries.DeleteDatabaseUser(ctx, id)
}

func (s *Store) QueueGrant(
	ctx context.Context, value databases.Grant, enabled bool, job jobs.Job,
) (databases.Grant, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return databases.Grant{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	var row dbgen.DatabaseGrant
	if enabled {
		row, err = q.UpsertDatabaseGrant(ctx, dbgen.UpsertDatabaseGrantParams{
			DatabaseID: value.DatabaseID, DatabaseUserID: value.UserID,
			CreatedAt: timeValue(value.CreatedAt), UpdatedAt: timeValue(value.UpdatedAt),
		})
	} else {
		row, err = q.QueueDatabaseGrantDelete(ctx, dbgen.QueueDatabaseGrantDeleteParams{
			UpdatedAt: timeValue(time.Now().UTC()), DatabaseID: value.DatabaseID, DatabaseUserID: value.UserID,
		})
	}
	if err != nil {
		return databases.Grant{}, jobs.Job{}, err
	}
	created, err := createJob(ctx, q, job)
	if err != nil {
		return databases.Grant{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return databases.Grant{}, jobs.Job{}, err
	}
	return databaseGrantValue(row), created, nil
}

func (s *Store) Grants(ctx context.Context, userID string, admin bool) ([]databases.Grant, error) {
	rows, err := s.queries.ListDatabaseGrants(ctx, dbgen.ListDatabaseGrantsParams{Column1: admin, UserID: userID})
	if err != nil {
		return nil, err
	}
	result := make([]databases.Grant, 0, len(rows))
	for _, row := range rows {
		result = append(result, databaseGrantValue(row))
	}
	return result, nil
}

func (s *Store) GrantPage(
	ctx context.Context, userID string, admin bool, query pagination.Query,
) (pagination.Result[databases.Grant], error) {
	total, err := s.queries.CountDatabaseGrants(ctx, dbgen.CountDatabaseGrantsParams{
		IsAdmin: admin, UserID: userID,
	})
	if err != nil {
		return pagination.Result[databases.Grant]{}, err
	}
	query = pagination.Clamp(query, total)
	rows, err := s.queries.ListDatabaseGrantsPage(ctx, dbgen.ListDatabaseGrantsPageParams{
		IsAdmin: admin, UserID: userID,
		PageOffset: pagination.Offset(query), PageSize: int64(query.Size),
	})
	if err != nil {
		return pagination.Result[databases.Grant]{}, err
	}
	items := make([]databases.Grant, 0, len(rows))
	for _, row := range rows {
		items = append(items, databaseGrantValue(row))
	}
	return pagination.Result[databases.Grant]{Items: items, Query: query, Total: total}, nil
}

func (s *Store) Grant(ctx context.Context, databaseID, userID string) (databases.Grant, error) {
	row, err := s.queries.GetDatabaseGrant(ctx, dbgen.GetDatabaseGrantParams{DatabaseID: databaseID, DatabaseUserID: userID})
	return databaseGrantValue(row), err
}

func (s *Store) SetGrantStatus(ctx context.Context, databaseID, userID, status string) error {
	return s.queries.UpdateDatabaseGrantStatus(ctx, dbgen.UpdateDatabaseGrantStatusParams{
		Status: status, UpdatedAt: timeValue(time.Now().UTC()), DatabaseID: databaseID, DatabaseUserID: userID,
	})
}

func (s *Store) DeleteGrant(ctx context.Context, databaseID, userID string) error {
	return s.queries.DeleteDatabaseGrant(ctx, dbgen.DeleteDatabaseGrantParams{DatabaseID: databaseID, DatabaseUserID: userID})
}

func createJob(ctx context.Context, q *dbgen.Queries, job jobs.Job) (jobs.Job, error) {
	row, err := q.CreateJob(ctx, dbgen.CreateJobParams{
		ID: job.ID, NodeID: nullString(job.NodeID), UserID: nullString(job.UserID), Kind: job.Kind,
		Payload: job.Payload, MaxAttempts: job.MaxAttempts, CreatedAt: timeValue(job.CreatedAt),
	})
	if err != nil {
		return jobs.Job{}, fmt.Errorf("create resource job: %w", err)
	}
	return jobValue(row), nil
}

func databaseValue(row dbgen.Database) databases.Database {
	return databases.Database{ID: row.ID, AccountID: row.AccountID, NodeID: row.NodeID, Name: row.Name, SystemName: row.SystemName, Status: row.Status, CreatedAt: timeFrom(row.CreatedAt), UpdatedAt: timeFrom(row.UpdatedAt)}
}

func databaseUserValue(row dbgen.DatabaseUser) databases.User {
	return databases.User{ID: row.ID, AccountID: row.AccountID, NodeID: row.NodeID, Name: row.Name, SystemName: row.SystemName, Status: row.Status, CreatedAt: timeFrom(row.CreatedAt), UpdatedAt: timeFrom(row.UpdatedAt)}
}

func databaseGrantValue(row dbgen.DatabaseGrant) databases.Grant {
	return databases.Grant{DatabaseID: row.DatabaseID, UserID: row.DatabaseUserID, Status: row.Status, CreatedAt: timeFrom(row.CreatedAt), UpdatedAt: timeFrom(row.UpdatedAt)}
}
