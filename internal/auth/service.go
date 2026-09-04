package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/GVALFER/WEBYCP/internal/idgen"
	"github.com/GVALFER/WEBYCP/internal/secret"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

const (
	sessionTTL  = 24 * time.Hour
	initialUser = "admin"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) InitAdmin(ctx context.Context) (Credentials, bool, error) {
	count, err := s.repository.UserCount(ctx)
	if err != nil {
		return Credentials{}, false, fmt.Errorf("count users: %w", err)
	}
	if count != 0 {
		return Credentials{}, false, nil
	}
	password, err := secret.Generate(24)
	if err != nil {
		return Credentials{}, false, err
	}
	user, passwordHash, err := prepareUser(
		initialUser, "Administrator", "admin@localhost.invalid", password, true,
	)
	if err != nil {
		return Credentials{}, false, err
	}
	created, err := s.repository.InitAdmin(ctx, newUser(user, passwordHash))
	if err != nil {
		return Credentials{}, false, err
	}
	if !created {
		return Credentials{}, false, nil
	}
	return Credentials{Username: user.Username, Password: password}, true, nil
}

func (s *Service) Login(ctx context.Context, username, password string) (Session, error) {
	normalized, err := validate.Username(username)
	if err != nil {
		_, _ = HashPassword(password)
		return Session{}, ErrInvalidCredentials
	}
	user, err := s.repository.UserByUsername(ctx, normalized)
	if err != nil {
		// Keep unknown users close to the same Argon2 cost as wrong passwords.
		_, _ = HashPassword(password)
		return Session{}, ErrInvalidCredentials
	}
	if !VerifyPassword(password, user.PasswordHash) {
		return Session{}, ErrInvalidCredentials
	}

	session, record, err := newSession(user.User)
	if err != nil {
		return Session{}, err
	}
	if err := s.repository.CreateSession(ctx, record); err != nil {
		return Session{}, fmt.Errorf("create session: %w", err)
	}

	return session, nil
}

func (s *Service) UpdateProfile(
	ctx context.Context,
	session Session,
	username, name, email, timezone, currentPassword, password string,
) (Session, error) {
	normalizedUsername, err := validate.Username(username)
	if err != nil {
		return Session{}, err
	}
	normalizedName, err := validate.Name(name)
	if err != nil {
		return Session{}, err
	}
	normalizedEmail, err := validate.Email(email)
	if err != nil {
		return Session{}, err
	}
	normalizedTimezone, err := validate.Timezone(timezone)
	if err != nil {
		return Session{}, err
	}
	record, err := s.repository.UserRecordByID(ctx, session.User.ID)
	if err != nil {
		return Session{}, ErrUnauthorized
	}
	passwordHash := ""
	if record.MustChangePassword && password == "" {
		return Session{}, &validate.Error{Field: "password", Message: "Choose a new password"}
	}
	if currentPassword != "" && password == "" {
		return Session{}, &validate.Error{Field: "password", Message: "Enter a new password"}
	}
	if password != "" {
		if err := validate.Password(password); err != nil {
			return Session{}, err
		}
		if record.MustChangePassword {
			if VerifyPassword(password, record.PasswordHash) {
				return Session{}, &validate.Error{Field: "password", Message: "Choose a different password"}
			}
		} else if !VerifyPassword(currentPassword, record.PasswordHash) {
			return Session{}, ErrCurrentPassword
		}
		passwordHash, err = HashPassword(password)
		if err != nil {
			return Session{}, err
		}
	}
	user, err := s.repository.UpdateProfile(ctx, UserUpdate{
		ID: session.User.ID, SessionID: session.ID,
		Username: normalizedUsername, Name: normalizedName, Email: normalizedEmail,
		Timezone:     normalizedTimezone,
		PasswordHash: passwordHash, UpdatedAt: time.Now().UTC(),
	})
	if err != nil {
		return Session{}, err
	}
	session.User = user
	return session, nil
}

func (s *Service) ResetPassword(ctx context.Context, username string) (Credentials, error) {
	normalized, err := validate.Username(username)
	if err != nil {
		return Credentials{}, err
	}
	password, err := secret.Generate(24)
	if err != nil {
		return Credentials{}, err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return Credentials{}, err
	}
	user, err := s.repository.ResetPassword(ctx, normalized, hash, time.Now().UTC())
	if err != nil {
		return Credentials{}, err
	}
	return Credentials{Username: user.Username, Password: password}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (Session, error) {
	if token == "" {
		return Session{}, ErrUnauthorized
	}

	record, err := s.repository.SessionByTokenHash(ctx, tokenHash(token), time.Now().UTC())
	if err != nil {
		return Session{}, ErrUnauthorized
	}
	user, err := s.repository.UserByID(ctx, record.UserID)
	if err != nil {
		return Session{}, ErrUnauthorized
	}

	return Session{
		ID:        record.ID,
		User:      user,
		CSRFToken: record.CSRFToken,
		ExpiresAt: record.ExpiresAt,
	}, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}

	return s.repository.DeleteSession(ctx, tokenHash(token))
}

func prepareUser(
	username, name, email, password string,
	mustChangePassword bool,
) (User, string, error) {
	normalizedUsername, err := validate.Username(username)
	if err != nil {
		return User{}, "", err
	}
	normalizedName, err := validate.Name(name)
	if err != nil {
		return User{}, "", err
	}
	normalizedEmail, err := validate.Email(email)
	if err != nil {
		return User{}, "", err
	}
	if err := validate.Password(password); err != nil {
		return User{}, "", err
	}

	passwordHash, err := HashPassword(password)
	if err != nil {
		return User{}, "", err
	}
	if err := validatePasswordHash(passwordHash); err != nil {
		return User{}, "", err
	}
	id, err := idgen.ID()
	if err != nil {
		return User{}, "", err
	}

	return User{
		ID: id, Username: normalizedUsername, Email: normalizedEmail,
		Name: normalizedName, Timezone: "UTC", Role: "admin", MustChangePassword: mustChangePassword,
		CreatedAt: time.Now().UTC(),
	}, passwordHash, nil
}

func newUser(user User, passwordHash string) NewUser {
	return NewUser{
		ID: user.ID, Username: user.Username, Email: user.Email, Name: user.Name,
		PasswordHash: passwordHash, Role: user.Role,
		MustChangePassword: user.MustChangePassword, CreatedAt: user.CreatedAt,
	}
}

func newSession(user User) (Session, NewSession, error) {
	id, err := idgen.ID()
	if err != nil {
		return Session{}, NewSession{}, err
	}
	token, err := idgen.Token()
	if err != nil {
		return Session{}, NewSession{}, err
	}
	csrfToken, err := idgen.Token()
	if err != nil {
		return Session{}, NewSession{}, err
	}

	now := time.Now().UTC()
	expiresAt := now.Add(sessionTTL)
	return Session{
		ID:        id,
		User:      user,
		Token:     token,
		CSRFToken: csrfToken,
		ExpiresAt: expiresAt,
	}, NewSession{
		ID:        id,
		UserID:    user.ID,
		TokenHash: tokenHash(token),
		CSRFToken: csrfToken,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}, nil
}

func tokenHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
