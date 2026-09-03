package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/GVALFER/WEBYCP/internal/idgen"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

const sessionTTL = 24 * time.Hour

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) BootstrapRequired(ctx context.Context) (bool, error) {
	count, err := s.repository.UserCount(ctx)
	if err != nil {
		return false, fmt.Errorf("count users: %w", err)
	}

	return count == 0, nil
}

func (s *Service) Bootstrap(ctx context.Context, name, email, password string) (Session, error) {
	user, passwordHash, err := prepareUser(name, email, password)
	if err != nil {
		return Session{}, err
	}

	session, record, err := newSession(user)
	if err != nil {
		return Session{}, err
	}
	if err := s.repository.Bootstrap(ctx, NewUser{
		ID:           user.ID,
		Email:        user.Email,
		Name:         user.Name,
		PasswordHash: passwordHash,
		Role:         user.Role,
		CreatedAt:    user.CreatedAt,
	}, record); err != nil {
		return Session{}, err
	}

	return session, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (Session, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	user, err := s.repository.UserByEmail(ctx, normalizedEmail)
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

func prepareUser(name, email, password string) (User, string, error) {
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
		ID:        id,
		Email:     normalizedEmail,
		Name:      normalizedName,
		Role:      "admin",
		CreatedAt: time.Now().UTC(),
	}, passwordHash, nil
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
