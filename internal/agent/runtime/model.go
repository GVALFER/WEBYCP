package runtime

import "context"

type Account struct {
	ID         string
	SystemUser string
	Home       string
	Version    string
}

type Pool struct {
	Socket string
}

type Driver interface {
	Ensure(context.Context, Account) (Pool, error)
}

type Cleaner interface {
	Delete(context.Context, string) error
}
