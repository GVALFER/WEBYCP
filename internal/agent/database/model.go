package database

import "context"

type Driver interface {
	EnsureDatabase(context.Context, string) error
	DeleteDatabase(context.Context, string) error
	EnsureUser(context.Context, string, string) error
	DeleteUser(context.Context, string) error
	EnsureGrant(context.Context, string, string) error
	DeleteGrant(context.Context, string, string) error
}
