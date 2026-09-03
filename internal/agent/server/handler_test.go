package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type accountManager struct {
	user string
}

func (m *accountManager) Ensure(_ context.Context, _, systemUser string) error {
	m.user = systemUser
	return nil
}

type domainManager struct {
	action    string
	accountID string
	user      string
	domainID  string
	name      string
	version   string
	aliases   []string
}

func (m *domainManager) Disable(
	_ context.Context,
	accountID, systemUser, domainID string,
) error {
	m.action = "disable"
	m.accountID = accountID
	m.user = systemUser
	m.domainID = domainID
	return nil
}

func (m *domainManager) Delete(
	_ context.Context,
	accountID, systemUser, domainID, name string,
) error {
	m.action = "delete"
	m.accountID = accountID
	m.user = systemUser
	m.domainID = domainID
	m.name = name
	return nil
}

func (m *domainManager) Rename(
	_ context.Context,
	accountID, systemUser, domainID, currentName, name, version string,
	aliases []string,
) error {
	m.action = "rename"
	m.accountID = accountID
	m.user = systemUser
	m.domainID = domainID
	m.name = currentName + ":" + name
	m.version = version
	m.aliases = aliases
	return nil
}

func (m *domainManager) Ensure(
	_ context.Context,
	accountID, systemUser, domainID, name, version string,
	aliases []string,
) error {
	m.accountID = accountID
	m.user = systemUser
	m.domainID = domainID
	m.name = name
	m.version = version
	m.aliases = aliases
	return nil
}

func TestHealth(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/agent/v1/health", nil)
	response := httptest.NewRecorder()

	New(Options{Version: "test"}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	if body := response.Body.String(); !strings.Contains(body, `"protocolVersion":"v1"`) {
		t.Fatalf("body = %q, want protocol version", body)
	}
}

func TestEnsureAccount(t *testing.T) {
	manager := &accountManager{}
	request := httptest.NewRequest(http.MethodPost, "/agent/v1/accounts", strings.NewReader(`{
		"accountId":"0123456789abcdef0123456789abcdef","systemUser":"wcp_0123456789ab"
	}`))
	response := httptest.NewRecorder()

	New(Options{Version: "test", Accounts: manager}).ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if manager.user != "wcp_0123456789ab" {
		t.Fatalf("system user = %q", manager.user)
	}
}

func TestEnsureDomain(t *testing.T) {
	manager := &domainManager{}
	request := httptest.NewRequest(http.MethodPost, "/agent/v1/domains", strings.NewReader(`{
		"accountId":"0123456789abcdef0123456789abcdef",
		"systemUser":"wcp_0123456789ab",
		"domainId":"fedcba9876543210fedcba9876543210",
		"name":"example.com",
		"phpVersion":"8.3",
		"aliases":["www.example.com","cdn.example.com"]
	}`))
	response := httptest.NewRecorder()

	New(Options{Version: "test", Domains: manager}).ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if manager.accountID != "0123456789abcdef0123456789abcdef" ||
		manager.user != "wcp_0123456789ab" ||
		manager.domainID != "fedcba9876543210fedcba9876543210" ||
		manager.name != "example.com" || manager.version != "8.3" ||
		len(manager.aliases) != 2 || manager.aliases[0] != "cdn.example.com" ||
		manager.aliases[1] != "www.example.com" {
		t.Fatalf("domain request = %+v", manager)
	}
}

func TestEnsureDomainRejectsUnsupportedPHPVersion(t *testing.T) {
	manager := &domainManager{}
	request := httptest.NewRequest(http.MethodPost, "/agent/v1/domains", strings.NewReader(`{
		"accountId":"0123456789abcdef0123456789abcdef",
		"systemUser":"wcp_0123456789ab",
		"domainId":"fedcba9876543210fedcba9876543210",
		"name":"example.com",
		"phpVersion":"8.4",
		"aliases":[]
	}`))
	response := httptest.NewRecorder()

	New(Options{Version: "test", Domains: manager}).ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if manager.domainID != "" {
		t.Fatal("domain manager should not run")
	}
}

func TestDomainLifecycle(t *testing.T) {
	manager := &domainManager{}
	handler := New(Options{Version: "test", Domains: manager})
	domainID := "fedcba9876543210fedcba9876543210"
	base := `{"accountId":"0123456789abcdef0123456789abcdef","systemUser":"wcp_0123456789ab","domainId":"` + domainID + `"}`
	disable := httptest.NewRequest(
		http.MethodPost, "/agent/v1/domains/disable", strings.NewReader(base),
	)
	disabled := httptest.NewRecorder()
	handler.ServeHTTP(disabled, disable)
	if disabled.Code != http.StatusNoContent || manager.action != "disable" {
		t.Fatalf("disable status = %d, action = %q", disabled.Code, manager.action)
	}

	remove := httptest.NewRequest(
		http.MethodPost, "/agent/v1/domains/delete",
		strings.NewReader(`{"accountId":"0123456789abcdef0123456789abcdef","systemUser":"wcp_0123456789ab","domainId":"`+domainID+`","name":"example.com"}`),
	)
	deleted := httptest.NewRecorder()
	handler.ServeHTTP(deleted, remove)
	if deleted.Code != http.StatusNoContent || manager.action != "delete" || manager.name != "example.com" {
		t.Fatalf("delete status = %d, manager = %+v", deleted.Code, manager)
	}

	rename := httptest.NewRequest(
		http.MethodPost, "/agent/v1/domains/rename",
		strings.NewReader(`{
            "accountId":"0123456789abcdef0123456789abcdef",
            "systemUser":"wcp_0123456789ab",
            "domainId":"`+domainID+`",
            "currentName":"example.com",
            "name":"renamed.example.com",
            "phpVersion":"8.3",
            "aliases":["www.example.com"]
        }`),
	)
	renamed := httptest.NewRecorder()
	handler.ServeHTTP(renamed, rename)
	if renamed.Code != http.StatusNoContent || manager.action != "rename" ||
		manager.name != "example.com:renamed.example.com" {
		t.Fatalf("rename status = %d, manager = %+v", renamed.Code, manager)
	}
}
