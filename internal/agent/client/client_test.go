package client

import (
	"context"
	"net/http"
	"testing"
	"time"

	agentserver "github.com/GVALFER/WEBYCP/internal/agent/server"
)

type manager struct {
	user string
}

func (m *manager) Ensure(_ context.Context, _, systemUser string) error {
	m.user = systemUser
	return nil
}

type domainManager struct {
	action  string
	name    string
	version string
	aliases []string
}

func (m *domainManager) Disable(_ context.Context, _, _, domainID string) error {
	m.action = "disable:" + domainID
	return nil
}

func (m *domainManager) Delete(_ context.Context, _, _, domainID, name string) error {
	m.action = "delete:" + domainID
	m.name = name
	return nil
}

func (m *domainManager) Rename(
	_ context.Context,
	_, _, domainID, currentName, name, version string,
	aliases []string,
) error {
	m.action = "rename:" + domainID
	m.name = currentName + ":" + name
	m.version = version
	m.aliases = aliases
	return nil
}

func (m *domainManager) Ensure(
	_ context.Context,
	_, _, _, name, version string,
	aliases []string,
) error {
	m.name = name
	m.version = version
	m.aliases = aliases
	return nil
}

func TestProbe(t *testing.T) {
	socket := t.TempDir() + "/agent.sock"
	listener, cleanup, err := agentserver.Listen(socket)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	server := &http.Server{Handler: agentserver.New(agentserver.Options{Version: "test"})}
	go func() { _ = server.Serve(listener) }()
	defer server.Shutdown(context.Background())

	if err := New(time.Second).Probe(context.Background(), socket); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureAccount(t *testing.T) {
	socket := t.TempDir() + "/agent.sock"
	listener, cleanup, err := agentserver.Listen(socket)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	accounts := &manager{}
	server := &http.Server{Handler: agentserver.New(agentserver.Options{
		Version: "test", Accounts: accounts,
	})}
	go func() { _ = server.Serve(listener) }()
	defer server.Shutdown(context.Background())

	err = New(time.Second).EnsureAccount(
		context.Background(), socket, "0123456789abcdef0123456789abcdef", "wcp_0123456789ab",
	)
	if err != nil {
		t.Fatal(err)
	}
	if accounts.user != "wcp_0123456789ab" {
		t.Fatalf("system user = %q", accounts.user)
	}
}

func TestEnsureDomain(t *testing.T) {
	socket := t.TempDir() + "/agent.sock"
	listener, cleanup, err := agentserver.Listen(socket)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	domains := &domainManager{}
	server := &http.Server{Handler: agentserver.New(agentserver.Options{
		Version: "test", Domains: domains,
	})}
	go func() { _ = server.Serve(listener) }()
	defer server.Shutdown(context.Background())

	err = New(time.Second).EnsureDomain(
		context.Background(), socket,
		"0123456789abcdef0123456789abcdef", "wcp_0123456789ab",
		"fedcba9876543210fedcba9876543210", "example.com", "8.3",
		[]string{"www.example.com"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if domains.name != "example.com" || domains.version != "8.3" ||
		len(domains.aliases) != 1 || domains.aliases[0] != "www.example.com" {
		t.Fatalf("domain request = %+v", domains)
	}
}

func TestDomainLifecycle(t *testing.T) {
	socket := t.TempDir() + "/agent.sock"
	listener, cleanup, err := agentserver.Listen(socket)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	domains := &domainManager{}
	server := &http.Server{Handler: agentserver.New(agentserver.Options{
		Version: "test", Domains: domains,
	})}
	go func() { _ = server.Serve(listener) }()
	defer server.Shutdown(context.Background())
	client := New(time.Second)
	domainID := "fedcba9876543210fedcba9876543210"

	if err := client.DisableDomain(
		context.Background(), socket, "0123456789abcdef0123456789abcdef",
		"wcp_0123456789ab", domainID,
	); err != nil {
		t.Fatal(err)
	}
	if domains.action != "disable:"+domainID {
		t.Fatalf("action = %q", domains.action)
	}
	if err := client.DeleteDomain(
		context.Background(), socket, "0123456789abcdef0123456789abcdef",
		"wcp_0123456789ab", domainID, "example.com",
	); err != nil {
		t.Fatal(err)
	}
	if domains.action != "delete:"+domainID || domains.name != "example.com" {
		t.Fatalf("delete request = %+v", domains)
	}
	if err := client.RenameDomain(
		context.Background(), socket, "0123456789abcdef0123456789abcdef",
		"wcp_0123456789ab", domainID, "example.com", "renamed.example.com", "8.3",
		[]string{"www.example.com"},
	); err != nil {
		t.Fatal(err)
	}
	if domains.action != "rename:"+domainID ||
		domains.name != "example.com:renamed.example.com" {
		t.Fatalf("rename request = %+v", domains)
	}
}

func TestProbeUnavailable(t *testing.T) {
	err := New(100*time.Millisecond).Probe(context.Background(), t.TempDir()+"/missing.sock")
	if err == nil {
		t.Fatal("expected probe error")
	}
}
