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

func (s *Store) InitAdmin(ctx context.Context, user auth.NewUser) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin administrator initialization: %w", err)
	}
	defer tx.Rollback()
	queries := s.queries.WithTx(tx)
	count, err := queries.CountUsers(ctx)
	if err != nil {
		return false, fmt.Errorf("count users: %w", err)
	}
	if count != 0 {
		return false, nil
	}
	if _, err := queries.CreateUser(ctx, userParams(user)); err != nil {
		return false, fmt.Errorf("create initial administrator: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit administrator initialization: %w", err)
	}
	return true, nil
}

func (s *Store) UserByUsername(ctx context.Context, username string) (auth.UserRecord, error) {
	user, err := s.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return auth.UserRecord{}, err
	}
	return userRecord(user), nil
}

func (s *Store) UserRecordByID(ctx context.Context, id string) (auth.UserRecord, error) {
	user, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		return auth.UserRecord{}, err
	}
	return userRecord(user), nil
}

func (s *Store) UpdateProfile(ctx context.Context, update auth.UserUpdate) (auth.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return auth.User{}, fmt.Errorf("begin profile update: %w", err)
	}
	defer tx.Rollback()
	queries := s.queries.WithTx(tx)
	usernameExists, err := queries.UsernameExistsExcept(ctx, dbgen.UsernameExistsExceptParams{
		Username: update.Username, ID: update.ID,
	})
	if err != nil {
		return auth.User{}, fmt.Errorf("check username: %w", err)
	}
	if usernameExists {
		return auth.User{}, auth.ErrUsernameExists
	}
	emailExists, err := queries.EmailExistsExcept(ctx, dbgen.EmailExistsExceptParams{
		Email: update.Email, ID: update.ID,
	})
	if err != nil {
		return auth.User{}, fmt.Errorf("check email: %w", err)
	}
	if emailExists {
		return auth.User{}, auth.ErrEmailExists
	}
	row, err := queries.UpdateUserProfile(ctx, dbgen.UpdateUserProfileParams{
		Username: update.Username, Name: update.Name, Email: update.Email,
		Timezone:  update.Timezone,
		UpdatedAt: timeValue(update.UpdatedAt), ID: update.ID,
	})
	if err != nil {
		return auth.User{}, fmt.Errorf("update profile: %w", err)
	}
	if update.PasswordHash != "" {
		row, err = queries.UpdateUserPassword(ctx, dbgen.UpdateUserPasswordParams{
			PasswordHash: update.PasswordHash, MustChangePassword: 0,
			UpdatedAt: timeValue(update.UpdatedAt), ID: update.ID,
		})
		if err != nil {
			return auth.User{}, fmt.Errorf("update password: %w", err)
		}
		if err := queries.DeleteOtherUserSessions(ctx, dbgen.DeleteOtherUserSessionsParams{
			UserID: update.ID, ID: update.SessionID,
		}); err != nil {
			return auth.User{}, fmt.Errorf("delete previous sessions: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return auth.User{}, fmt.Errorf("commit profile update: %w", err)
	}
	return userValue(row), nil
}

func (s *Store) ResetPassword(
	ctx context.Context, username, passwordHash string, updatedAt time.Time,
) (auth.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return auth.User{}, fmt.Errorf("begin password reset: %w", err)
	}
	defer tx.Rollback()
	queries := s.queries.WithTx(tx)
	user, err := queries.GetUserByUsername(ctx, username)
	if err != nil {
		return auth.User{}, err
	}
	user, err = queries.UpdateUserPassword(ctx, dbgen.UpdateUserPasswordParams{
		PasswordHash: passwordHash, MustChangePassword: 1,
		UpdatedAt: timeValue(updatedAt), ID: user.ID,
	})
	if err != nil {
		return auth.User{}, fmt.Errorf("reset password: %w", err)
	}
	if err := queries.DeleteUserSessions(ctx, user.ID); err != nil {
		return auth.User{}, fmt.Errorf("delete user sessions: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return auth.User{}, fmt.Errorf("commit password reset: %w", err)
	}
	return userValue(user), nil
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
		ID: user.ID, Username: user.Username, Email: user.Email, Name: user.Name,
		PasswordHash: user.PasswordHash, Role: user.Role,
		MustChangePassword: boolValue(user.MustChangePassword),
		CreatedAt:          timeValue(user.CreatedAt), UpdatedAt: timeValue(user.CreatedAt),
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
		ID: user.ID, Username: user.Username, Email: user.Email, Name: user.Name,
		Timezone: user.Timezone,
		Role:     user.Role, MustChangePassword: user.MustChangePassword != 0,
		CreatedAt: timeFrom(user.CreatedAt),
	}
}

func userRecord(user dbgen.User) auth.UserRecord {
	return auth.UserRecord{
		User:         userValue(user),
		PasswordHash: user.PasswordHash,
	}
}
