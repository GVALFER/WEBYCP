package auth

import (
	"context"
	"errors"
	"time"
)

var (
	ErrBootstrapComplete  = errors.New("bootstrap is already complete")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
)

type User struct {
	ID        string
	Email     string
	Name      string
	Role      string
	CreatedAt time.Time
}

type UserRecord struct {
	User
	PasswordHash string
}

type Session struct {
	ID        string
	User      User
	Token     string
	CSRFToken string
	ExpiresAt time.Time
}

type SessionRecord struct {
	ID        string
	UserID    string
	TokenHash string
	CSRFToken string
	ExpiresAt time.Time
}

type NewUser struct {
	ID           string
	Email        string
	Name         string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
}

type NewSession struct {
	ID        string
	UserID    string
	TokenHash string
	CSRFToken string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type Repository interface {
	UserCount(context.Context) (int64, error)
	Bootstrap(context.Context, NewUser, NewSession) error
	UserByEmail(context.Context, string) (UserRecord, error)
	UserByID(context.Context, string) (User, error)
	CreateSession(context.Context, NewSession) error
	SessionByTokenHash(context.Context, string, time.Time) (SessionRecord, error)
	DeleteSession(context.Context, string) error
}
