package backupfmt

import "time"

const Version = 1

type Entry struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
}

type Manifest struct {
	Version   int       `json:"version"`
	RunID     string    `json:"runId"`
	AccountID string    `json:"accountId"`
	CreatedAt time.Time `json:"createdAt"`
	Files     bool      `json:"files"`
	Databases bool      `json:"databases"`
	Metadata  bool      `json:"metadata"`
	Entries   []Entry   `json:"entries"`
}

type Metadata struct {
	Version   int        `json:"version"`
	AccountID string     `json:"accountId"`
	Domains   []Domain   `json:"domains"`
	Databases []Database `json:"databases"`
	CronJobs  []CronJob  `json:"cronJobs"`
}

type Domain struct {
	ID         string  `json:"id"`
	NodeID     string  `json:"nodeId"`
	Name       string  `json:"name"`
	PHPVersion string  `json:"phpVersion"`
	Enabled    bool    `json:"enabled"`
	Aliases    []Alias `json:"aliases"`
}

type Alias struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type Database struct {
	ID         string `json:"id"`
	NodeID     string `json:"nodeId"`
	Name       string `json:"name"`
	SystemName string `json:"systemName"`
}

type CronJob struct {
	ID       string `json:"id"`
	NodeID   string `json:"nodeId"`
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
	Command  string `json:"command"`
	Enabled  bool   `json:"enabled"`
}
