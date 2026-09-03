package webserver

import "context"

type Site struct {
	ID            string
	Name          string
	Aliases       []string
	Root          string
	PHPSocket     string
	Certificate   string
	Key           string
	RedirectHTTPS bool
}

type Driver interface {
	Ensure(context.Context, Site) error
	Disable(context.Context, string) error
	Delete(context.Context, string) error
}
