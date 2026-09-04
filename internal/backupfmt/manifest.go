package backupfmt

import "time"

const Version = 2

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
	Websites  []Website  `json:"websites"`
	Databases []Database `json:"databases"`
	CronJobs  []CronJob  `json:"cronJobs"`
}

type Website struct {
	ID             string          `json:"id"`
	NodeID         string          `json:"nodeId"`
	Name           string          `json:"name"`
	Kind           string          `json:"kind"`
	DocumentRoot   string          `json:"documentRoot"`
	WebDriver      string          `json:"webDriver"`
	RuntimeDriver  string          `json:"runtimeDriver"`
	RuntimeVersion string          `json:"runtimeVersion"`
	Enabled        bool            `json:"enabled"`
	Domains        []WebsiteDomain `json:"domains"`
}

type WebsiteDomain struct {
	ID       string `json:"id"`
	Hostname string `json:"hostname"`
	Kind     string `json:"kind"`
	Enabled  bool   `json:"enabled"`
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
