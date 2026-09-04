package client

import (
	"context"
	"net/http"
	"testing"
	"time"

	agentserver "github.com/GVALFER/WEBYCP/internal/agent/server"
	agentwebsite "github.com/GVALFER/WEBYCP/internal/agent/website"
	"github.com/GVALFER/WEBYCP/internal/services"
	"github.com/GVALFER/WEBYCP/internal/websites"
)

type manager struct{ user string }

func (m *manager) Ensure(_ context.Context, _, user string) error { m.user = user; return nil }

type websiteManager struct {
	action string
	spec   agentwebsite.Spec
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

func TestProbe(t *testing.T) {
	socket, server := testServer(t, agentserver.Options{
		Version: "test", Capabilities: capabilityObserver{},
	})
	defer server.Shutdown(context.Background())
	value, err := New(time.Second).Probe(context.Background(), socket)
	if err != nil {
		t.Fatal(err)
	}
	if len(value.Webservers) != 1 || value.Webservers[0].Driver != services.Nginx {
		t.Fatalf("capabilities = %+v", value)
	}
}

func TestEnsureAccount(t *testing.T) {
	accounts := &manager{}
	socket, server := testServer(t, agentserver.Options{Version: "test", Accounts: accounts})
	defer server.Shutdown(context.Background())
	if err := New(time.Second).EnsureAccount(context.Background(), socket, "0123456789abcdef0123456789abcdef", "wcp_0123456789ab"); err != nil {
		t.Fatal(err)
	}
	if accounts.user != "wcp_0123456789ab" {
		t.Fatalf("system user = %q", accounts.user)
	}
}

func TestWebsiteLifecycle(t *testing.T) {
	manager := &websiteManager{}
	socket, server := testServer(t, agentserver.Options{Version: "test", Websites: manager})
	defer server.Shutdown(context.Background())
	spec := websites.Spec{AccountID: "0123456789abcdef0123456789abcdef", SystemUser: "wcp_0123456789ab", WebsiteID: "fedcba9876543210fedcba9876543210", DocumentRoot: "/home/wcp_0123456789ab/web/fedcba9876543210fedcba9876543210/public_html", Kind: "php", WebDriver: "nginx", RuntimeDriver: "phpfpm", RuntimeVersion: "8.3", PrimaryDomain: "example.com", Aliases: []string{"www.example.com"}}
	client := New(time.Second)
	for _, item := range []struct {
		action string
		run    func(context.Context, string, websites.Spec) error
	}{{"ensure", client.EnsureWebsite}, {"disable", client.DisableWebsite}, {"delete", client.DeleteWebsite}} {
		if err := item.run(context.Background(), socket, spec); err != nil {
			t.Fatal(err)
		}
		if manager.action != item.action || manager.spec.DocumentRoot != spec.DocumentRoot {
			t.Fatalf("manager = %+v", manager)
		}
	}
}

func TestProbeUnavailable(t *testing.T) {
	if _, err := New(100*time.Millisecond).Probe(context.Background(), t.TempDir()+"/missing.sock"); err == nil {
		t.Fatal("expected probe error")
	}
}

type capabilityObserver struct{}

func (capabilityObserver) Observe(context.Context) services.Capabilities {
	return services.Capabilities{
		Webservers: []services.Capability{{Driver: services.Nginx, Status: services.Healthy}},
	}
}

func testServer(t *testing.T, options agentserver.Options) (string, *http.Server) {
	t.Helper()
	socket := t.TempDir() + "/agent.sock"
	listener, cleanup, err := agentserver.Listen(socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	server := &http.Server{Handler: agentserver.New(options)}
	go func() { _ = server.Serve(listener) }()
	return socket, server
}
