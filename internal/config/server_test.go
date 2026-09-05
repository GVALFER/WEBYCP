package config

import "testing"

func TestProductionDefaults(t *testing.T) {
	for _, key := range []string{
		"WEBYCP_SERVER_ADDR",
		"WEBYCP_DATABASE_PATH",
		"WEBYCP_AGENT_SOCKET",
		"WEBYCP_SECURE_COOKIE",
	} {
		t.Setenv(key, "")
	}

	server := ServerFromEnv()
	if server.Addr != defaultServerAddr ||
		server.DatabasePath != defaultDatabasePath ||
		server.AgentSocket != defaultAgentSocket ||
		!server.SecureCookie {
		t.Fatalf("ServerFromEnv() = %+v, want production defaults", server)
	}
	if agent := AgentFromEnv(); agent.Socket != defaultAgentSocket ||
		agent.PowerDNSURL != defaultPowerDNSURL || agent.PowerDNSKeyPath != defaultPowerDNSKey {
		t.Fatalf("AgentFromEnv() = %+v", agent)
	}
}

func TestEnvironmentOverrides(t *testing.T) {
	t.Setenv("WEBYCP_SERVER_ADDR", "127.0.0.1:9000")
	t.Setenv("WEBYCP_DATABASE_PATH", "/tmp/webycp.db")
	t.Setenv("WEBYCP_AGENT_SOCKET", "/tmp/webycp.sock")
	t.Setenv("WEBYCP_SECURE_COOKIE", "false")

	server := ServerFromEnv()
	if server.Addr != "127.0.0.1:9000" ||
		server.DatabasePath != "/tmp/webycp.db" ||
		server.AgentSocket != "/tmp/webycp.sock" ||
		server.SecureCookie {
		t.Fatalf("ServerFromEnv() = %+v, want environment overrides", server)
	}
	if agent := AgentFromEnv(); agent.Socket != "/tmp/webycp.sock" {
		t.Fatalf("AgentFromEnv().Socket = %q, want environment override", agent.Socket)
	}
}
