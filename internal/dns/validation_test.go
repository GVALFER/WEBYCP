package dns

import (
	"testing"
)

func TestNormalizeRecord(t *testing.T) {
	tests := []struct {
		name    string
		record  Record
		want    Record
		invalid bool
	}{
		{
			name:   "apex IPv4",
			record: Record{Name: "@", Type: "a", Content: "192.0.2.1", TTL: 3600},
			want:   Record{Name: "example.test", Type: "A", Content: "192.0.2.1", TTL: 3600},
		},
		{
			name:   "relative MX",
			record: Record{Name: "mail", Type: "MX", Content: "MX.EXAMPLE.TEST.", TTL: 600, Priority: 10},
			want:   Record{Name: "mail.example.test", Type: "MX", Content: "mx.example.test", TTL: 600, Priority: 10},
		},
		{
			name:   "service TXT",
			record: Record{Name: "_dmarc", Type: "TXT", Content: "v=DMARC1; p=reject", TTL: 300},
			want:   Record{Name: "_dmarc.example.test", Type: "TXT", Content: "v=DMARC1; p=reject", TTL: 300},
		},
		{
			name:    "apex CNAME",
			record:  Record{Name: "@", Type: "CNAME", Content: "target.example.test", TTL: 300},
			invalid: true,
		},
		{
			name:    "outside zone",
			record:  Record{Name: "www.other.test", Type: "A", Content: "192.0.2.1", TTL: 300},
			invalid: true,
		},
		{
			name:    "unicode label",
			record:  Record{Name: "café", Type: "A", Content: "192.0.2.1", TTL: 300},
			invalid: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, err := NormalizeRecord(test.record, "example.test")
			if test.invalid {
				if err == nil {
					t.Fatalf("record = %+v, want error", value)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if value.Name != test.want.Name || value.Type != test.want.Type ||
				value.Content != test.want.Content || value.TTL != test.want.TTL ||
				value.Priority != test.want.Priority {
				t.Fatalf("record = %+v, want %+v", value, test.want)
			}
		})
	}
}

func TestValidateSettings(t *testing.T) {
	value, err := ValidateSettings(Settings{
		PrimaryNameserver: "NS1.Example.COM.", SecondaryNameserver: "ns2.example.com",
		DefaultTTL: 3600,
	})
	if err != nil {
		t.Fatal(err)
	}
	if value.PrimaryNameserver != "ns1.example.com" {
		t.Fatalf("primary nameserver = %q", value.PrimaryNameserver)
	}
	if _, err := ValidateSettings(Settings{
		PrimaryNameserver: "ns1.example.com", SecondaryNameserver: "ns1.example.com",
		DefaultTTL: 3600,
	}); err == nil {
		t.Fatal("duplicate nameservers were accepted")
	}
}

func TestRecordSetsRetainLastSyncedIdentity(t *testing.T) {
	current := Record{
		ID: "record-1", Name: "new.example.test", Type: "AAAA",
		Content: "2001:db8::1", TTL: 3600,
		SyncedName: "old.example.test", SyncedType: "A", Status: "pending",
	}
	records := []Record{
		current,
		{ID: "record-2", Name: "new.example.test", Type: "AAAA", Content: "2001:db8::2", TTL: 3600, Status: "active"},
	}
	sets := recordSets(records, current, recordPayload{RecordID: current.ID})
	if len(sets) != 2 || sets[0].Name != "old.example.test" || len(sets[0].Records) != 0 ||
		sets[1].Name != "new.example.test" || len(sets[1].Records) != 2 {
		t.Fatalf("record sets = %+v", sets)
	}
}
