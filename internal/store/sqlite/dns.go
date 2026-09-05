package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/GVALFER/WEBYCP/internal/dns"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite/dbgen"
)

func (s *Store) EnsureDNSProvider(ctx context.Context, value dns.Provider) (dns.Provider, error) {
	row, err := s.queries.EnsureDNSProvider(ctx, dbgen.EnsureDNSProviderParams{
		ID: value.ID, NodeID: value.NodeID, Name: value.Name, Driver: value.Driver,
		CreatedAt: timeValue(value.CreatedAt), UpdatedAt: timeValue(value.UpdatedAt),
	})
	return dnsProviderValue(row), err
}

func (s *Store) DNSProviders(ctx context.Context) ([]dns.Provider, error) {
	rows, err := s.queries.ListDNSProviders(ctx)
	if err != nil {
		return nil, err
	}
	values := make([]dns.Provider, 0, len(rows))
	for _, row := range rows {
		values = append(values, dnsProviderValue(row))
	}
	return values, nil
}

func (s *Store) DNSProvider(ctx context.Context, id string) (dns.Provider, error) {
	row, err := s.queries.GetDNSProvider(ctx, id)
	return dnsProviderValue(row), err
}

func (s *Store) DNSSettings(ctx context.Context) (dns.Settings, error) {
	row, err := s.queries.GetDNSSettings(ctx)
	return dnsSettingsValue(row), err
}

func (s *Store) UpdateDNSSettings(ctx context.Context, value dns.Settings) (dns.Settings, error) {
	row, err := s.queries.UpdateDNSSettings(ctx, dbgen.UpdateDNSSettingsParams{
		PrimaryNameserver:   value.PrimaryNameserver,
		SecondaryNameserver: value.SecondaryNameserver,
		DefaultTtl:          value.DefaultTTL,
		UpdatedAt:           timeValue(value.UpdatedAt),
	})
	return dnsSettingsValue(row), err
}

func (s *Store) CreateDNSZoneProvision(
	ctx context.Context, zone dns.Zone, job jobs.Job,
) (dns.Zone, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return dns.Zone{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	row, err := q.CreateDNSZone(ctx, dbgen.CreateDNSZoneParams{
		ID: zone.ID, AccountID: zone.AccountID, NodeID: zone.NodeID,
		ProviderID: zone.ProviderID, Name: zone.Name,
		PrimaryNameserver: zone.PrimaryNameserver, SecondaryNameserver: zone.SecondaryNameserver,
		CreatedAt: timeValue(zone.CreatedAt), UpdatedAt: timeValue(zone.UpdatedAt),
	})
	if err != nil {
		if constraint(err) {
			return dns.Zone{}, jobs.Job{}, dns.ErrNameExists
		}
		return dns.Zone{}, jobs.Job{}, err
	}
	createdJob, err := createJob(ctx, q, job)
	if err != nil {
		return dns.Zone{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return dns.Zone{}, jobs.Job{}, err
	}
	return dnsZoneValue(row), createdJob, nil
}

func (s *Store) DNSZone(ctx context.Context, id string) (dns.Zone, error) {
	row, err := s.queries.GetDNSZone(ctx, id)
	return dnsZoneValue(row), err
}

func (s *Store) DNSZonePage(
	ctx context.Context, userID string, admin bool, query pagination.Query,
) (pagination.Result[dns.Zone], error) {
	var total int64
	var rows []dbgen.DnsZone
	var err error
	if admin {
		total, err = s.queries.CountDNSZones(ctx)
	} else {
		total, err = s.queries.CountUserDNSZones(ctx, userID)
	}
	if err != nil {
		return pagination.Result[dns.Zone]{}, err
	}
	query = pagination.Clamp(query, total)
	if admin {
		rows, err = s.queries.ListDNSZonesPage(ctx, dbgen.ListDNSZonesPageParams{
			PageSize: int64(query.Size), PageOffset: pagination.Offset(query),
		})
	} else {
		rows, err = s.queries.ListUserDNSZonesPage(ctx, dbgen.ListUserDNSZonesPageParams{
			UserID: userID, PageSize: int64(query.Size), PageOffset: pagination.Offset(query),
		})
	}
	if err != nil {
		return pagination.Result[dns.Zone]{}, err
	}
	return pagination.Result[dns.Zone]{Items: dnsZoneValues(rows), Query: query, Total: total}, nil
}

func (s *Store) QueueDNSZoneDelete(
	ctx context.Context, id string, job jobs.Job,
) (dns.Zone, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return dns.Zone{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	row, err := q.QueueDNSZoneDelete(ctx, dbgen.QueueDNSZoneDeleteParams{
		UpdatedAt: timeValue(time.Now().UTC()), ID: id,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return dns.Zone{}, jobs.Job{}, dns.ErrBusy
	}
	if err != nil {
		return dns.Zone{}, jobs.Job{}, err
	}
	createdJob, err := createJob(ctx, q, job)
	if err != nil {
		return dns.Zone{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return dns.Zone{}, jobs.Job{}, err
	}
	return dnsZoneValue(row), createdJob, nil
}

func (s *Store) UpdateDNSZoneStatus(ctx context.Context, id, status string) error {
	return s.queries.UpdateDNSZoneStatus(ctx, dbgen.UpdateDNSZoneStatusParams{
		Status: status, UpdatedAt: timeValue(time.Now().UTC()), ID: id,
	})
}

func (s *Store) DeleteDNSZone(ctx context.Context, id string) error {
	return s.queries.DeleteDNSZone(ctx, id)
}

func (s *Store) CreateDNSRecordProvision(
	ctx context.Context, record dns.Record, job jobs.Job,
) (dns.Record, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return dns.Record{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	if err := ensureDNSRecordAvailable(ctx, q, record, ""); err != nil {
		return dns.Record{}, jobs.Job{}, err
	}
	row, err := q.CreateDNSRecord(ctx, dbgen.CreateDNSRecordParams{
		ID: record.ID, ZoneID: record.ZoneID, Name: record.Name, Type: record.Type,
		Content: record.Content, Ttl: record.TTL, Priority: record.Priority,
		CreatedAt: timeValue(record.CreatedAt), UpdatedAt: timeValue(record.UpdatedAt),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return dns.Record{}, jobs.Job{}, dns.ErrBusy
	}
	if err != nil {
		if constraint(err) {
			return dns.Record{}, jobs.Job{}, dns.ErrRecordExists
		}
		return dns.Record{}, jobs.Job{}, err
	}
	createdJob, err := createJob(ctx, q, job)
	if err != nil {
		return dns.Record{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return dns.Record{}, jobs.Job{}, err
	}
	return dnsRecordValue(row), createdJob, nil
}

func (s *Store) DNSRecord(ctx context.Context, id string) (dns.Record, error) {
	row, err := s.queries.GetDNSRecord(ctx, id)
	return dnsRecordValue(row), err
}

func (s *Store) DNSRecordPage(
	ctx context.Context, zoneID string, query pagination.Query,
) (pagination.Result[dns.Record], error) {
	total, err := s.queries.CountDNSRecords(ctx, zoneID)
	if err != nil {
		return pagination.Result[dns.Record]{}, err
	}
	query = pagination.Clamp(query, total)
	rows, err := s.queries.ListDNSRecordsPage(ctx, dbgen.ListDNSRecordsPageParams{
		ZoneID: zoneID, PageSize: int64(query.Size), PageOffset: pagination.Offset(query),
	})
	if err != nil {
		return pagination.Result[dns.Record]{}, err
	}
	return pagination.Result[dns.Record]{Items: dnsRecordValues(rows), Query: query, Total: total}, nil
}

func (s *Store) DNSRecords(ctx context.Context, zoneID string) ([]dns.Record, error) {
	rows, err := s.queries.ListDNSRecords(ctx, zoneID)
	return dnsRecordValues(rows), err
}

func (s *Store) QueueDNSRecordUpdate(
	ctx context.Context, record dns.Record, job jobs.Job,
) (dns.Record, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return dns.Record{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	if err := ensureDNSRecordAvailable(ctx, q, record, record.ID); err != nil {
		return dns.Record{}, jobs.Job{}, err
	}
	row, err := q.QueueDNSRecordUpdate(ctx, dbgen.QueueDNSRecordUpdateParams{
		Name: record.Name, Type: record.Type, Content: record.Content,
		Ttl: record.TTL, Priority: record.Priority,
		UpdatedAt: timeValue(record.UpdatedAt), ID: record.ID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return dns.Record{}, jobs.Job{}, dns.ErrBusy
	}
	if err != nil {
		if constraint(err) {
			return dns.Record{}, jobs.Job{}, dns.ErrRecordExists
		}
		return dns.Record{}, jobs.Job{}, err
	}
	createdJob, err := createJob(ctx, q, job)
	if err != nil {
		return dns.Record{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return dns.Record{}, jobs.Job{}, err
	}
	return dnsRecordValue(row), createdJob, nil
}

func (s *Store) QueueDNSRecordDelete(
	ctx context.Context, id string, job jobs.Job,
) (dns.Record, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return dns.Record{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	row, err := q.QueueDNSRecordDelete(ctx, dbgen.QueueDNSRecordDeleteParams{
		UpdatedAt: timeValue(time.Now().UTC()), ID: id,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return dns.Record{}, jobs.Job{}, dns.ErrBusy
	}
	if err != nil {
		return dns.Record{}, jobs.Job{}, err
	}
	createdJob, err := createJob(ctx, q, job)
	if err != nil {
		return dns.Record{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return dns.Record{}, jobs.Job{}, err
	}
	return dnsRecordValue(row), createdJob, nil
}

func (s *Store) UpdateDNSRecordStatus(ctx context.Context, id, status string) error {
	return s.queries.UpdateDNSRecordStatus(ctx, dbgen.UpdateDNSRecordStatusParams{
		Status: status, UpdatedAt: timeValue(time.Now().UTC()), ID: id,
	})
}

func (s *Store) CompleteDNSRecordSync(ctx context.Context, id string) error {
	return s.queries.CompleteDNSRecordSync(ctx, dbgen.CompleteDNSRecordSyncParams{
		UpdatedAt: timeValue(time.Now().UTC()), ID: id,
	})
}

func (s *Store) DeleteDNSRecord(ctx context.Context, id string) error {
	return s.queries.DeleteDNSRecord(ctx, id)
}

func ensureDNSRecordAvailable(
	ctx context.Context, q *dbgen.Queries, value dns.Record, excludeID string,
) error {
	rows, err := q.ListDNSRecordsByName(ctx, dbgen.ListDNSRecordsByNameParams{
		ZoneID: value.ZoneID, Name: value.Name,
	})
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.ID == excludeID {
			continue
		}
		if row.Type == value.Type && row.Content == value.Content && row.Priority == value.Priority {
			return dns.ErrRecordExists
		}
		if row.Type == "CNAME" || value.Type == "CNAME" {
			return dns.ErrRecordConflict
		}
		if row.Type == value.Type && row.Ttl != value.TTL {
			return dns.ErrRecordConflict
		}
	}
	return nil
}

func constraint(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func dnsProviderValue(row dbgen.DnsProvider) dns.Provider {
	return dns.Provider{
		ID: row.ID, NodeID: row.NodeID, Name: row.Name, Driver: row.Driver,
		CreatedAt: timeFrom(row.CreatedAt), UpdatedAt: timeFrom(row.UpdatedAt),
	}
}

func dnsSettingsValue(row dbgen.DnsSetting) dns.Settings {
	return dns.Settings{
		PrimaryNameserver:   row.PrimaryNameserver,
		SecondaryNameserver: row.SecondaryNameserver,
		DefaultTTL:          row.DefaultTtl,
		UpdatedAt:           timeFrom(row.UpdatedAt),
	}
}

func dnsZoneValues(rows []dbgen.DnsZone) []dns.Zone {
	values := make([]dns.Zone, 0, len(rows))
	for _, row := range rows {
		values = append(values, dnsZoneValue(row))
	}
	return values
}

func dnsZoneValue(row dbgen.DnsZone) dns.Zone {
	return dns.Zone{
		ID: row.ID, AccountID: row.AccountID, NodeID: row.NodeID,
		ProviderID: row.ProviderID, Name: row.Name, Status: row.Status,
		PrimaryNameserver: row.PrimaryNameserver, SecondaryNameserver: row.SecondaryNameserver,
		CreatedAt: timeFrom(row.CreatedAt), UpdatedAt: timeFrom(row.UpdatedAt),
	}
}

func dnsRecordValues(rows []dbgen.DnsRecord) []dns.Record {
	values := make([]dns.Record, 0, len(rows))
	for _, row := range rows {
		values = append(values, dnsRecordValue(row))
	}
	return values
}

func dnsRecordValue(row dbgen.DnsRecord) dns.Record {
	return dns.Record{
		ID: row.ID, ZoneID: row.ZoneID, Name: row.Name, Type: row.Type,
		Content: row.Content, SyncedName: row.SyncedName, SyncedType: row.SyncedType,
		TTL: row.Ttl, Priority: row.Priority, Status: row.Status,
		CreatedAt: timeFrom(row.CreatedAt), UpdatedAt: timeFrom(row.UpdatedAt),
	}
}
