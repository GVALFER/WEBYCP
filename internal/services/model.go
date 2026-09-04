package services

import (
	"context"
	"time"
)

const (
	Nginx   = "nginx"
	PHPFPM  = "phpfpm"
	PHP83   = "8.3"
	MySQL   = "mysql"
	Crontab = "crontab"
	Local   = "local"

	Healthy     = "healthy"
	Unavailable = "unavailable"
)

type Capability struct {
	Driver  string
	Version string
	Status  string
}

type Capabilities struct {
	Webservers []Capability
	Runtimes   []Capability
	Databases  []Capability
	Schedulers []Capability
	Backups    []Capability
}

type Defaults struct {
	WebDriver       string
	RuntimeDriver   string
	RuntimeVersion  string
	DatabaseDriver  string
	SchedulerDriver string
	BackupDriver    string
}

type Settings struct {
	Defaults  Defaults
	UpdatedAt time.Time
}

type Repository interface {
	ServiceSettings(context.Context) (Settings, error)
	UpdateServiceSettings(context.Context, Settings) (Settings, error)
}
