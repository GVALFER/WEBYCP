package certificate

import (
	"context"
	"time"
)

type Request struct {
	ID, Kind, WebsiteID, AccountID, SystemUser string
	Name, Email, DocumentRoot, RuntimeVersion  string
	Names                                      []string
	RedirectHTTPS                              bool
}

type Result struct {
	Names     []string
	ExpiresAt time.Time
}

type Driver interface {
	Issue(context.Context, Request) (Result, error)
}
