package sqlite

import (
	"errors"
	"testing"
	"time"

	"github.com/GVALFER/WEBYCP/internal/dns"
	"github.com/GVALFER/WEBYCP/internal/jobs"
)

func TestDNSRecordSetConstraints(t *testing.T) {
	ctx, store, account := limitStore(t)
	now := time.Now().UTC()
	provider, err := store.EnsureDNSProvider(ctx, dns.Provider{
		ID: limitID(5000), NodeID: account.NodeID, Name: "Local PowerDNS",
		Driver: dns.PowerDNS, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	zone := dns.Zone{
		ID: limitID(5001), AccountID: account.ID, NodeID: account.NodeID,
		ProviderID: provider.ID, Name: "example.test", CreatedAt: now, UpdatedAt: now,
	}
	if _, _, err := store.CreateDNSZoneProvision(
		ctx, zone, limitJob(5001, account.NodeID, jobs.KindDNSZoneCreate),
	); err != nil {
		t.Fatal(err)
	}
	duplicate := zone
	duplicate.ID = limitID(5002)
	if _, _, err := store.CreateDNSZoneProvision(
		ctx, duplicate, limitJob(5002, account.NodeID, jobs.KindDNSZoneCreate),
	); !errors.Is(err, dns.ErrNameExists) {
		t.Fatalf("duplicate zone error = %v", err)
	}
	if err := store.UpdateDNSZoneStatus(ctx, zone.ID, "active"); err != nil {
		t.Fatal(err)
	}

	first := dns.Record{
		ID: limitID(5101), ZoneID: zone.ID, Name: "www.example.test", Type: "A",
		Content: "192.0.2.1", TTL: 3600, CreatedAt: now, UpdatedAt: now,
	}
	if _, _, err := store.CreateDNSRecordProvision(
		ctx, first, limitJob(5101, account.NodeID, jobs.KindDNSRecordSync),
	); err != nil {
		t.Fatal(err)
	}

	differentTTL := first
	differentTTL.ID, differentTTL.Content, differentTTL.TTL = limitID(5102), "192.0.2.2", 600
	if _, _, err := store.CreateDNSRecordProvision(
		ctx, differentTTL, limitJob(5102, account.NodeID, jobs.KindDNSRecordSync),
	); !errors.Is(err, dns.ErrRecordConflict) {
		t.Fatalf("mixed TTL error = %v", err)
	}

	cname := first
	cname.ID, cname.Type, cname.Content = limitID(5103), "CNAME", "target.example.test"
	if _, _, err := store.CreateDNSRecordProvision(
		ctx, cname, limitJob(5103, account.NodeID, jobs.KindDNSRecordSync),
	); !errors.Is(err, dns.ErrRecordConflict) {
		t.Fatalf("CNAME conflict error = %v", err)
	}

	second := first
	second.ID, second.Content = limitID(5104), "192.0.2.2"
	if _, _, err := store.CreateDNSRecordProvision(
		ctx, second, limitJob(5104, account.NodeID, jobs.KindDNSRecordSync),
	); err != nil {
		t.Fatalf("same-TTL A record: %v", err)
	}
}
