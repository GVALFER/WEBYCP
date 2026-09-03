package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite/dbgen"
)

func (s *Store) UserCount(ctx context.Context) (int64, error) {
	return s.queries.CountUsers(ctx)
}

func (s *Store) Bootstrap(ctx context.Context, user auth.NewUser, session auth.NewSession) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin bootstrap: %w", err)
	}
	defer tx.Rollback()

	queries := s.queries.WithTx(tx)
	count, err := queries.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("count bootstrap users: %w", err)
	}
	if count != 0 {
		return auth.ErrBootstrapComplete
	}
	if _, err := queries.CreateUser(ctx, userParams(user)); err != nil {
		return fmt.Errorf("create bootstrap user: %w", err)
	}
	if _, err := queries.CreateSession(ctx, sessionParams(session)); err != nil {
		return fmt.Errorf("create bootstrap session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit bootstrap: %w", err)
	}

	return nil
}

func (s *Store) UserByEmail(ctx context.Context, email string) (auth.UserRecord, error) {
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return auth.UserRecord{}, err
	}
	return userRecord(user), nil
}

func (s *Store) UserByID(ctx context.Context, id string) (auth.User, error) {
	user, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		return auth.User{}, err
	}
	return userValue(user), nil
}

func (s *Store) CreateSession(ctx context.Context, session auth.NewSession) error {
	_, err := s.queries.CreateSession(ctx, sessionParams(session))
	return err
}

func (s *Store) SessionByTokenHash(ctx context.Context, hash string, now time.Time) (auth.SessionRecord, error) {
	session, err := s.queries.GetSessionByTokenHash(ctx, dbgen.GetSessionByTokenHashParams{
		TokenHash: hash,
		ExpiresAt: timeValue(now),
	})
	if err != nil {
		return auth.SessionRecord{}, err
	}
	return auth.SessionRecord{
		ID:        session.ID,
		UserID:    session.UserID,
		TokenHash: session.TokenHash,
		CSRFToken: session.CsrfToken,
		ExpiresAt: timeFrom(session.ExpiresAt),
	}, nil
}

func (s *Store) DeleteSession(ctx context.Context, hash string) error {
	return s.queries.DeleteSessionByTokenHash(ctx, hash)
}

func userParams(user auth.NewUser) dbgen.CreateUserParams {
	return dbgen.CreateUserParams{
		ID:           user.ID,
		Email:        user.Email,
		Name:         user.Name,
		PasswordHash: user.PasswordHash,
		Role:         user.Role,
		CreatedAt:    timeValue(user.CreatedAt),
		UpdatedAt:    timeValue(user.CreatedAt),
	}
}

func sessionParams(session auth.NewSession) dbgen.CreateSessionParams {
	return dbgen.CreateSessionParams{
		ID:        session.ID,
		UserID:    session.UserID,
		TokenHash: session.TokenHash,
		CsrfToken: session.CSRFToken,
		ExpiresAt: timeValue(session.ExpiresAt),
		CreatedAt: timeValue(session.CreatedAt),
	}
}

func userValue(user dbgen.User) auth.User {
	return auth.User{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		CreatedAt: timeFrom(user.CreatedAt),
	}
}

func userRecord(user dbgen.User) auth.UserRecord {
	return auth.UserRecord{
		User:         userValue(user),
		PasswordHash: user.PasswordHash,
	}
}
