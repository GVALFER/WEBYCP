package dns

import (
	"context"
	"errors"
	"time"

	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/pagination"
)

const PowerDNS = "powerdns"

var (
	ErrBusy           = errors.New("DNS resource operation is pending")
	ErrNameExists     = errors.New("DNS zone name already exists")
	ErrRecordExists   = errors.New("DNS record already exists")
	ErrRecordConflict = errors.New("DNS record conflicts with the existing record set")
	ErrNotConfigured  = errors.New("DNS nameservers are not configured")
)

type Provider struct {
	ID, NodeID, Name, Driver, Status string
	CreatedAt, UpdatedAt             time.Time
}

type Settings struct {
	PrimaryNameserver, SecondaryNameserver string
	DefaultTTL                             int64
	UpdatedAt                              time.Time
}

type Zone struct {
	ID, AccountID, NodeID, ProviderID, Name, Status string
	PrimaryNameserver, SecondaryNameserver          string
	CreatedAt, UpdatedAt                            time.Time
}

type Record struct {
	ID, ZoneID, Name, Type, Content, Status string
	SyncedName, SyncedType                  string
	TTL, Priority                           int64
	CreatedAt, UpdatedAt                    time.Time
}

type RecordSet struct {
	Name, Type string
	TTL        int64
	Records    []string
}

type Repository interface {
	EnsureDNSProvider(context.Context, Provider) (Provider, error)
	DNSProviders(context.Context) ([]Provider, error)
	DNSProvider(context.Context, string) (Provider, error)
	DNSSettings(context.Context) (Settings, error)
	UpdateDNSSettings(context.Context, Settings) (Settings, error)
	CreateDNSZoneProvision(context.Context, Zone, jobs.Job) (Zone, jobs.Job, error)
	DNSZone(context.Context, string) (Zone, error)
	DNSZonePage(context.Context, string, bool, pagination.Query) (pagination.Result[Zone], error)
	QueueDNSZoneDelete(context.Context, string, jobs.Job) (Zone, jobs.Job, error)
	UpdateDNSZoneStatus(context.Context, string, string) error
	DeleteDNSZone(context.Context, string) error
	CreateDNSRecordProvision(context.Context, Record, jobs.Job) (Record, jobs.Job, error)
	DNSRecord(context.Context, string) (Record, error)
	DNSRecordPage(context.Context, string, pagination.Query) (pagination.Result[Record], error)
	DNSRecords(context.Context, string) ([]Record, error)
	QueueDNSRecordUpdate(context.Context, Record, jobs.Job) (Record, jobs.Job, error)
	QueueDNSRecordDelete(context.Context, string, jobs.Job) (Record, jobs.Job, error)
	UpdateDNSRecordStatus(context.Context, string, string) error
	CompleteDNSRecordSync(context.Context, string) error
	DeleteDNSRecord(context.Context, string) error
}

type Agent interface {
	EnsureDNSZone(context.Context, string, ZoneSpec) error
	DeleteDNSZone(context.Context, string, ZoneSpec) error
	SyncDNSRecordSets(context.Context, string, ZoneSpec, []RecordSet) error
}

type ZoneSpec struct {
	ID, Name    string
	Nameservers []string
}
