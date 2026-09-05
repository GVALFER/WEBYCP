package ftp

import "context"

// Entry is a virtual login, not a Unix identity. Its home is derived by the Agent.
type Entry struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"passwordHash"`
	Enabled      bool   `json:"enabled"`
}

type Driver interface {
	Sync(context.Context, string, string, []Entry) error
}
