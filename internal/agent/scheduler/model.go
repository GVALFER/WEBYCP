package scheduler

import "context"

type Entry struct {
	ID, Schedule, Command string
}

type Driver interface {
	Sync(context.Context, string, string, []Entry) error
}
