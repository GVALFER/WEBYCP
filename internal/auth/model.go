package auth

import (
	"context"
	"errors"
	"time"
)

var (
	ErrUsernameExists     = errors.New("username already exists")
	ErrEmailExists        = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrCurrentPassword    = errors.New("current password is incorrect")
	ErrUnauthorized       = errors.New("unauthorized")
)

type User struct {
	ID                 string
	Username           string
	Email              string
	Name               string
	Timezone           string
	Role               string
	MustChangePassword bool
	CreatedAt          time.Time
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
	ID                 string
	Username           string
	Email              string
	Name               string
	PasswordHash       string
	Role               string
	MustChangePassword bool
	CreatedAt          time.Time
}

type UserUpdate struct {
	ID           string
	SessionID    string
	Username     string
	Email        string
	Name         string
	Timezone     string
	PasswordHash string
	UpdatedAt    time.Time
}

type Credentials struct {
	Username string
	Password string
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
	InitAdmin(context.Context, NewUser) (bool, error)
	UserByUsername(context.Context, string) (UserRecord, error)
	UserByID(context.Context, string) (User, error)
	UserRecordByID(context.Context, string) (UserRecord, error)
	UpdateProfile(context.Context, UserUpdate) (User, error)
	ResetPassword(context.Context, string, string, time.Time) (User, error)
	CreateSession(context.Context, NewSession) error
	SessionByTokenHash(context.Context, string, time.Time) (SessionRecord, error)
	DeleteSession(context.Context, string) error
}
