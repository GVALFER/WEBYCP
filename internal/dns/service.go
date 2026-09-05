package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/idgen"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/nodes"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

type Service struct {
	repository Repository
	accounts   *accounts.Service
	nodes      nodes.Repository
	agent      Agent
	notify     func()
}

type zonePayload struct {
	ZoneID string `json:"zoneId"`
}

type recordPayload struct {
	RecordID string `json:"recordId"`
	Delete   bool   `json:"delete,omitempty"`
}

func NewService(
	repository Repository,
	accountService *accounts.Service,
	nodeRepository nodes.Repository,
	agent Agent,
	notify func(),
) *Service {
	return &Service{
		repository: repository, accounts: accountService, nodes: nodeRepository,
		agent: agent, notify: notify,
	}
}

func (s *Service) EnsureLocalProvider(
	ctx context.Context, nodeID string,
) (Provider, error) {
	id, err := idgen.ID()
	if err != nil {
		return Provider{}, err
	}
	now := time.Now().UTC()
	return s.repository.EnsureDNSProvider(ctx, Provider{
		ID: id, NodeID: nodeID, Name: "Local PowerDNS", Driver: PowerDNS,
		CreatedAt: now, UpdatedAt: now,
	})
}

func (s *Service) Providers(ctx context.Context) ([]Provider, error) {
	values, err := s.repository.DNSProviders(ctx)
	if err != nil {
		return nil, err
	}
	for index := range values {
		values[index].Status = "unknown"
		node, nodeErr := s.nodes.Node(ctx, values[index].NodeID)
		if nodeErr != nil || node.Capabilities == nil {
			continue
		}
		for _, capability := range node.Capabilities.DNS {
			if capability.Driver == values[index].Driver {
				values[index].Status = capability.Status
				break
			}
		}
	}
	return values, nil
}

func (s *Service) Settings(ctx context.Context) (Settings, error) {
	return s.repository.DNSSettings(ctx)
}

func (s *Service) UpdateSettings(ctx context.Context, value Settings) (Settings, error) {
	value, err := ValidateSettings(value)
	if err != nil {
		return Settings{}, err
	}
	value.UpdatedAt = time.Now().UTC()
	return s.repository.UpdateDNSSettings(ctx, value)
}

func (s *Service) ZonePage(
	ctx context.Context, userID string, admin bool, query pagination.Query,
) (pagination.Result[Zone], error) {
	return s.repository.DNSZonePage(ctx, userID, admin, query)
}

func (s *Service) Zone(
	ctx context.Context, id, userID string, admin bool,
) (Zone, error) {
	zone, err := s.repository.DNSZone(ctx, id)
	if err != nil {
		return Zone{}, err
	}
	if _, err := s.accounts.Account(ctx, zone.AccountID, userID, admin); err != nil {
		return Zone{}, err
	}
	return zone, nil
}

func (s *Service) CreateZone(
	ctx context.Context, accountID, providerID, name, userID string, admin bool,
) (Zone, jobs.Job, error) {
	account, err := s.accounts.Account(ctx, accountID, userID, admin)
	if err != nil {
		return Zone{}, jobs.Job{}, err
	}
	if account.Status != "active" || !account.Enabled {
		return Zone{}, jobs.Job{}, accounts.ErrBusy
	}
	provider, err := s.repository.DNSProvider(ctx, providerID)
	if err != nil {
		return Zone{}, jobs.Job{}, fmt.Errorf("get DNS provider: %w", err)
	}
	if provider.Driver != PowerDNS || provider.NodeID != account.NodeID {
		return Zone{}, jobs.Job{}, &validate.Error{
			Field: "providerId", Message: "Choose a DNS provider on the hosting account server",
		}
	}
	settings, err := s.repository.DNSSettings(ctx)
	if err != nil {
		return Zone{}, jobs.Job{}, err
	}
	if settings.PrimaryNameserver == "" || settings.SecondaryNameserver == "" {
		return Zone{}, jobs.Job{}, ErrNotConfigured
	}
	name, err = validate.Domain(name)
	if err != nil {
		return Zone{}, jobs.Job{}, err
	}
	zoneID, err := idgen.ID()
	if err != nil {
		return Zone{}, jobs.Job{}, err
	}
	job, err := dnsJob(account.NodeID, userID, jobs.KindDNSZoneCreate, zonePayload{ZoneID: zoneID})
	if err != nil {
		return Zone{}, jobs.Job{}, err
	}
	now := time.Now().UTC()
	zone := Zone{
		ID: zoneID, AccountID: account.ID, NodeID: account.NodeID,
		ProviderID: provider.ID, Name: name, Status: "pending",
		PrimaryNameserver:   settings.PrimaryNameserver,
		SecondaryNameserver: settings.SecondaryNameserver,
		CreatedAt:           now, UpdatedAt: now,
	}
	zone, job, err = s.repository.CreateDNSZoneProvision(ctx, zone, job)
	if err != nil {
		return Zone{}, jobs.Job{}, err
	}
	s.notify()
	return zone, job, nil
}

func (s *Service) DeleteZone(
	ctx context.Context, id, userID string, admin bool,
) (Zone, jobs.Job, error) {
	zone, err := s.Zone(ctx, id, userID, admin)
	if err != nil {
		return Zone{}, jobs.Job{}, err
	}
	if zone.Status == "pending" || zone.Status == "deleting" {
		return Zone{}, jobs.Job{}, ErrBusy
	}
	job, err := dnsJob(zone.NodeID, userID, jobs.KindDNSZoneDelete, zonePayload{ZoneID: zone.ID})
	if err != nil {
		return Zone{}, jobs.Job{}, err
	}
	zone, job, err = s.repository.QueueDNSZoneDelete(ctx, zone.ID, job)
	if err == nil {
		s.notify()
	}
	return zone, job, err
}

func (s *Service) RecordPage(
	ctx context.Context, zoneID, userID string, admin bool, query pagination.Query,
) (pagination.Result[Record], error) {
	if _, err := s.Zone(ctx, zoneID, userID, admin); err != nil {
		return pagination.Result[Record]{}, err
	}
	return s.repository.DNSRecordPage(ctx, zoneID, query)
}

func (s *Service) CreateRecord(
	ctx context.Context, zoneID, name, kind, content string, ttl, priority int64,
	userID string, admin bool,
) (Record, jobs.Job, error) {
	zone, err := s.Zone(ctx, zoneID, userID, admin)
	if err != nil {
		return Record{}, jobs.Job{}, err
	}
	if zone.Status != "active" {
		return Record{}, jobs.Job{}, ErrBusy
	}
	record, err := NormalizeRecord(Record{
		ZoneID: zone.ID, Name: name, Type: kind, Content: content,
		TTL: ttl, Priority: priority,
	}, zone.Name)
	if err != nil {
		return Record{}, jobs.Job{}, err
	}
	record.ID, err = idgen.ID()
	if err != nil {
		return Record{}, jobs.Job{}, err
	}
	job, err := dnsJob(zone.NodeID, userID, jobs.KindDNSRecordSync, recordPayload{RecordID: record.ID})
	if err != nil {
		return Record{}, jobs.Job{}, err
	}
	now := time.Now().UTC()
	record.Status, record.CreatedAt, record.UpdatedAt = "pending", now, now
	record, job, err = s.repository.CreateDNSRecordProvision(ctx, record, job)
	if err != nil {
		return Record{}, jobs.Job{}, err
	}
	s.notify()
	return record, job, nil
}

func (s *Service) UpdateRecord(
	ctx context.Context, id, name, kind, content string, ttl, priority int64,
	userID string, admin bool,
) (Record, jobs.Job, error) {
	current, zone, err := s.record(ctx, id, userID, admin)
	if err != nil {
		return Record{}, jobs.Job{}, err
	}
	if current.Status == "pending" || current.Status == "deleting" || zone.Status != "active" {
		return Record{}, jobs.Job{}, ErrBusy
	}
	value, err := NormalizeRecord(Record{
		ID: current.ID, ZoneID: current.ZoneID, Name: name, Type: kind,
		Content: content, TTL: ttl, Priority: priority,
		CreatedAt: current.CreatedAt, UpdatedAt: time.Now().UTC(),
	}, zone.Name)
	if err != nil {
		return Record{}, jobs.Job{}, err
	}
	payload := recordPayload{RecordID: current.ID}
	job, err := dnsJob(zone.NodeID, userID, jobs.KindDNSRecordSync, payload)
	if err != nil {
		return Record{}, jobs.Job{}, err
	}
	value, job, err = s.repository.QueueDNSRecordUpdate(ctx, value, job)
	if err == nil {
		s.notify()
	}
	return value, job, err
}

func (s *Service) DeleteRecord(
	ctx context.Context, id, userID string, admin bool,
) (Record, jobs.Job, error) {
	record, zone, err := s.record(ctx, id, userID, admin)
	if err != nil {
		return Record{}, jobs.Job{}, err
	}
	if record.Status == "pending" || record.Status == "deleting" || zone.Status != "active" {
		return Record{}, jobs.Job{}, ErrBusy
	}
	payload := recordPayload{RecordID: record.ID, Delete: true}
	job, err := dnsJob(zone.NodeID, userID, jobs.KindDNSRecordSync, payload)
	if err != nil {
		return Record{}, jobs.Job{}, err
	}
	record, job, err = s.repository.QueueDNSRecordDelete(ctx, record.ID, job)
	if err == nil {
		s.notify()
	}
	return record, job, err
}

func (s *Service) ProvisionZone(ctx context.Context, job jobs.Job) error {
	var payload zonePayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil || payload.ZoneID == "" {
		return fmt.Errorf("decode DNS zone job payload")
	}
	zone, err := s.repository.DNSZone(ctx, payload.ZoneID)
	if err != nil {
		return fmt.Errorf("get DNS zone: %w", err)
	}
	provider, node, spec, err := s.zoneSpec(ctx, zone)
	if err != nil {
		return err
	}
	if provider.Driver != PowerDNS {
		return fmt.Errorf("unsupported DNS provider %q", provider.Driver)
	}
	switch job.Kind {
	case jobs.KindDNSZoneDelete:
		if err := s.agent.DeleteDNSZone(ctx, node.Endpoint, spec); err != nil {
			_ = s.repository.UpdateDNSZoneStatus(ctx, zone.ID, "error")
			return fmt.Errorf("delete DNS zone: %w", err)
		}
		return s.repository.DeleteDNSZone(ctx, zone.ID)
	case jobs.KindDNSZoneCreate:
	default:
		return fmt.Errorf("unsupported DNS zone job kind %q", job.Kind)
	}
	if err := s.repository.UpdateDNSZoneStatus(ctx, zone.ID, "pending"); err != nil {
		return err
	}
	if err := s.agent.EnsureDNSZone(ctx, node.Endpoint, spec); err != nil {
		_ = s.repository.UpdateDNSZoneStatus(ctx, zone.ID, "error")
		return fmt.Errorf("ensure DNS zone: %w", err)
	}
	return s.repository.UpdateDNSZoneStatus(ctx, zone.ID, "active")
}

func (s *Service) ProvisionRecord(ctx context.Context, job jobs.Job) error {
	var payload recordPayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil || payload.RecordID == "" {
		return fmt.Errorf("decode DNS record job payload")
	}
	record, err := s.repository.DNSRecord(ctx, payload.RecordID)
	if err != nil {
		return fmt.Errorf("get DNS record: %w", err)
	}
	zone, err := s.repository.DNSZone(ctx, record.ZoneID)
	if err != nil {
		return fmt.Errorf("get DNS zone: %w", err)
	}
	provider, node, spec, err := s.zoneSpec(ctx, zone)
	if err != nil {
		return err
	}
	if provider.Driver != PowerDNS {
		return fmt.Errorf("unsupported DNS provider %q", provider.Driver)
	}
	records, err := s.repository.DNSRecords(ctx, zone.ID)
	if err != nil {
		return fmt.Errorf("list DNS records: %w", err)
	}
	if !payload.Delete {
		if err := s.repository.UpdateDNSRecordStatus(ctx, record.ID, "pending"); err != nil {
			return err
		}
	}
	sets := recordSets(records, record, payload)
	if err := s.agent.SyncDNSRecordSets(ctx, node.Endpoint, spec, sets); err != nil {
		_ = s.repository.UpdateDNSRecordStatus(ctx, record.ID, "error")
		return fmt.Errorf("sync DNS records: %w", err)
	}
	if payload.Delete {
		return s.repository.DeleteDNSRecord(ctx, record.ID)
	}
	return s.repository.CompleteDNSRecordSync(ctx, record.ID)
}

func (s *Service) record(
	ctx context.Context, id, userID string, admin bool,
) (Record, Zone, error) {
	record, err := s.repository.DNSRecord(ctx, id)
	if err != nil {
		return Record{}, Zone{}, err
	}
	zone, err := s.Zone(ctx, record.ZoneID, userID, admin)
	return record, zone, err
}

func (s *Service) zoneSpec(
	ctx context.Context, zone Zone,
) (Provider, nodes.Node, ZoneSpec, error) {
	provider, err := s.repository.DNSProvider(ctx, zone.ProviderID)
	if err != nil {
		return Provider{}, nodes.Node{}, ZoneSpec{}, fmt.Errorf("get DNS provider: %w", err)
	}
	node, err := s.nodes.Node(ctx, zone.NodeID)
	if err != nil {
		return Provider{}, nodes.Node{}, ZoneSpec{}, fmt.Errorf("get DNS node: %w", err)
	}
	return provider, node, ZoneSpec{
		ID: zone.ID, Name: zone.Name,
		Nameservers: []string{zone.PrimaryNameserver, zone.SecondaryNameserver},
	}, nil
}

func dnsJob(nodeID, userID, kind string, value any) (jobs.Job, error) {
	id, err := idgen.ID()
	if err != nil {
		return jobs.Job{}, err
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return jobs.Job{}, err
	}
	return jobs.Job{
		ID: id, NodeID: nodeID, UserID: userID, Kind: kind, Status: "queued",
		Payload: string(payload), MaxAttempts: 2, CreatedAt: time.Now().UTC(),
	}, nil
}

func recordSets(records []Record, current Record, payload recordPayload) []RecordSet {
	keys := make([][2]string, 0, 2)
	if current.SyncedName != "" &&
		(current.SyncedName != current.Name || current.SyncedType != current.Type) {
		keys = append(keys, [2]string{current.SyncedName, current.SyncedType})
	}
	keys = append(keys, [2]string{current.Name, current.Type})
	sets := make([]RecordSet, 0, len(keys))
	for _, key := range keys {
		set := RecordSet{Name: key[0], Type: key[1]}
		for _, record := range records {
			if record.Name != key[0] || record.Type != key[1] || record.Status == "deleting" ||
				(payload.Delete && record.ID == payload.RecordID) {
				continue
			}
			set.TTL = record.TTL
			set.Records = append(set.Records, ProviderContent(record))
		}
		sort.Strings(set.Records)
		sets = append(sets, set)
	}
	return sets
}
