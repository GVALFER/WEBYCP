package powerdns

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	agentdns "github.com/GVALFER/WEBYCP/internal/agent/dns"
)

const testZoneID = "0123456789abcdef0123456789abcdef"

func TestEnsureZone(t *testing.T) {
	var request zone
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "secret" {
			t.Fatal("missing PowerDNS API key")
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := New(server.URL, "secret")
	err := client.EnsureZone(context.Background(), agentdns.Zone{
		ID: testZoneID, Name: "example.test",
		Nameservers: []string{"ns1.example.test", "ns2.example.test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if request.Name != "example.test." || request.Account != "webycp:"+testZoneID ||
		request.SOAEditAPI != "DEFAULT" || len(request.RRSets) != 2 ||
		request.RRSets[0].Type != "SOA" ||
		!strings.HasPrefix(request.RRSets[0].Records[0].Content,
			"ns1.example.test. hostmaster.example.test. ") ||
		request.RRSets[1].Type != "NS" ||
		request.RRSets[1].Records[1].Content != "ns2.example.test." {
		t.Fatalf("request = %+v", request)
	}
}

func TestEnsureZoneIsIdempotentOnlyForOwnedZone(t *testing.T) {
	account := "webycp:" + testZoneID
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusConflict)
			return
		}
		_ = json.NewEncoder(w).Encode(zone{Account: account})
	}))
	defer server.Close()
	client := New(server.URL, "secret")
	value := agentdns.Zone{
		ID: testZoneID, Name: "example.test",
		Nameservers: []string{"ns1.example.test", "ns2.example.test"},
	}
	if err := client.EnsureZone(context.Background(), value); err != nil {
		t.Fatal(err)
	}
	account = "someone-else"
	if err := client.EnsureZone(context.Background(), value); err == nil {
		t.Fatal("foreign zone was accepted")
	}
}

func TestSyncRecordSets(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(zone{Account: "webycp:" + testZoneID})
			return
		}
		var body struct {
			RRSets []rrset `json:"rrsets"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.RRSets) != 2 || body.RRSets[0].ChangeType != "DELETE" ||
			body.RRSets[1].ChangeType != "REPLACE" || body.RRSets[1].TTL != 3600 {
			t.Fatalf("rrsets = %+v", body.RRSets)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	client := New(server.URL, "secret")
	err := client.SyncRecordSets(context.Background(), agentdns.Zone{
		ID: testZoneID, Name: "example.test",
	}, []agentdns.RecordSet{
		{Name: "old.example.test", Type: "A"},
		{Name: "www.example.test", Type: "A", TTL: 3600, Records: []string{"192.0.2.1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d", requests)
	}
}

func TestErrorsDoNotExposeAPIKey(t *testing.T) {
	const key = "do-not-log-this-key"
	client := New("http://127.0.0.1:1", key)
	err := client.Health(context.Background())
	if err == nil || strings.Contains(err.Error(), key) {
		t.Fatalf("error = %v", err)
	}
}
