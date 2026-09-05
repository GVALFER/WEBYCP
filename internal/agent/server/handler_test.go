package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	agentdns "github.com/GVALFER/WEBYCP/internal/agent/dns"
	agentwebsite "github.com/GVALFER/WEBYCP/internal/agent/website"
	"github.com/GVALFER/WEBYCP/internal/services"
)

type accountManager struct{ user string }

type capabilityObserver struct{}

func (capabilityObserver) Observe(context.Context) services.Capabilities {
	return services.Capabilities{
		Webservers: []services.Capability{{Driver: services.Nginx, Status: services.Healthy}},
	}
}

func (m *accountManager) Ensure(_ context.Context, _, user string) error { m.user = user; return nil }

type websiteManager struct {
	action string
	spec   agentwebsite.Spec
}

type dnsDriver struct {
	action string
	zone   agentdns.Zone
	sets   []agentdns.RecordSet
}

func (d *dnsDriver) Health(context.Context) error { return nil }
func (d *dnsDriver) EnsureZone(_ context.Context, zone agentdns.Zone) error {
	d.action, d.zone = "ensure", zone
	return nil
}
func (d *dnsDriver) DeleteZone(_ context.Context, zone agentdns.Zone) error {
	d.action, d.zone = "delete", zone
	return nil
}
func (d *dnsDriver) SyncRecordSets(
	_ context.Context, zone agentdns.Zone, sets []agentdns.RecordSet,
) error {
	d.action, d.zone, d.sets = "sync", zone, sets
	return nil
}

func (m *websiteManager) Ensure(_ context.Context, spec agentwebsite.Spec) error {
	m.action, m.spec = "ensure", spec
	return nil
}
func (m *websiteManager) Disable(_ context.Context, spec agentwebsite.Spec) error {
	m.action, m.spec = "disable", spec
	return nil
}
func (m *websiteManager) Delete(_ context.Context, spec agentwebsite.Spec) error {
	m.action, m.spec = "delete", spec
	return nil
}

func TestHealth(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/agent/v1/health", nil)
	response := httptest.NewRecorder()
	New(Options{Version: "test", Capabilities: capabilityObserver{}}).ServeHTTP(response, request)
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"protocolVersion":"v1"`) ||
		!strings.Contains(response.Body.String(), `"driver":"nginx"`) {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestHealthRequiresCapabilityObserver(t *testing.T) {
	response := httptest.NewRecorder()
	New(Options{Version: "test"}).ServeHTTP(
		response, httptest.NewRequest(http.MethodGet, "/agent/v1/health", nil),
	)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestEnsureAccount(t *testing.T) {
	manager := &accountManager{}
	request := httptest.NewRequest(http.MethodPost, "/agent/v1/accounts", strings.NewReader(`{"accountId":"0123456789abcdef0123456789abcdef","systemUser":"wcp_0123456789ab"}`))
	response := httptest.NewRecorder()
	New(Options{Version: "test", Accounts: manager}).ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || manager.user != "wcp_0123456789ab" {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestWebsiteLifecycle(t *testing.T) {
	manager := &websiteManager{}
	handler := New(Options{Version: "test", Websites: manager})
	body := `{
        "accountId":"0123456789abcdef0123456789abcdef",
        "systemUser":"wcp_0123456789ab",
        "websiteId":"fedcba9876543210fedcba9876543210",
        "documentRoot":"/home/wcp_0123456789ab/web/fedcba9876543210fedcba9876543210/public_html",
        "kind":"php",
        "webDriver":"nginx",
        "runtimeDriver":"phpfpm",
        "runtimeVersion":"8.3",
        "primaryDomain":"example.com",
        "aliases":["www.example.com"]
    }`
	for _, item := range []struct{ path, action string }{{"/agent/v1/websites", "ensure"}, {"/agent/v1/websites/disable", "disable"}, {"/agent/v1/websites/delete", "delete"}} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, item.path, strings.NewReader(body)))
		if response.Code != http.StatusNoContent || manager.action != item.action || manager.spec.PrimaryDomain != "example.com" {
			t.Fatalf("%s status = %d, body = %s, manager = %+v", item.action, response.Code, response.Body.String(), manager)
		}
	}
}

func TestWebsiteRejectsUnknownDriver(t *testing.T) {
	response := httptest.NewRecorder()
	New(Options{Websites: &websiteManager{}}).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/agent/v1/websites", strings.NewReader(`{"accountId":"a","systemUser":"b","websiteId":"c","documentRoot":"/tmp","kind":"php","webDriver":"apache","runtimeDriver":"phpfpm","runtimeVersion":"8.3","primaryDomain":"example.com","aliases":[]}`)))
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestDNSLifecycle(t *testing.T) {
	driver := &dnsDriver{}
	handler := New(Options{DNS: driver})
	zone := `{
        "id":"0123456789abcdef0123456789abcdef",
        "name":"example.test",
        "nameservers":["ns1.example.test","ns2.example.test"]
    }`
	for _, item := range []struct {
		method string
		action string
	}{
		{method: http.MethodPost, action: "ensure"},
		{method: http.MethodDelete, action: "delete"},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(item.method, "/agent/v1/dns/zones", strings.NewReader(zone)))
		if response.Code != http.StatusNoContent || driver.action != item.action || driver.zone.Name != "example.test" {
			t.Fatalf("%s status = %d, body = %s", item.action, response.Code, response.Body.String())
		}
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/agent/v1/dns/records", strings.NewReader(`{
        "zone":`+zone+`,
        "rrsets":[{"name":"www.example.test","type":"A","ttl":3600,"records":["192.0.2.1"]}]
    }`)))
	if response.Code != http.StatusNoContent || driver.action != "sync" || len(driver.sets) != 1 {
		t.Fatalf("sync status = %d, body = %s, driver = %+v", response.Code, response.Body.String(), driver)
	}
}
