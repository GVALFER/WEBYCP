package config

import "testing"

func TestProductionDefaults(t *testing.T) {
	for _, key := range []string{
		"WEBYCP_SERVER_ADDR",
		"WEBYCP_DATABASE_PATH",
		"WEBYCP_AGENT_SOCKET",
		"WEBYCP_WEB_DIR",
		"WEBYCP_SECURE_COOKIE",
	} {
		t.Setenv(key, "")
	}

	server := ServerFromEnv()
	if server.Addr != defaultServerAddr ||
		server.DatabasePath != defaultDatabasePath ||
		server.AgentSocket != defaultAgentSocket ||
		server.WebDir != defaultWebDir ||
		!server.SecureCookie {
		t.Fatalf("ServerFromEnv() = %+v, want production defaults", server)
	}
	if agent := AgentFromEnv(); agent.Socket != defaultAgentSocket {
		t.Fatalf("AgentFromEnv().Socket = %q, want %q", agent.Socket, defaultAgentSocket)
	}
}

func TestEnvironmentOverrides(t *testing.T) {
	t.Setenv("WEBYCP_SERVER_ADDR", "127.0.0.1:9000")
	t.Setenv("WEBYCP_DATABASE_PATH", "/tmp/webycp.db")
	t.Setenv("WEBYCP_AGENT_SOCKET", "/tmp/webycp.sock")
	t.Setenv("WEBYCP_WEB_DIR", "/tmp/web")
	t.Setenv("WEBYCP_SECURE_COOKIE", "false")

	server := ServerFromEnv()
	if server.Addr != "127.0.0.1:9000" ||
		server.DatabasePath != "/tmp/webycp.db" ||
		server.AgentSocket != "/tmp/webycp.sock" ||
		server.WebDir != "/tmp/web" ||
		server.SecureCookie {
		t.Fatalf("ServerFromEnv() = %+v, want environment overrides", server)
	}
	if agent := AgentFromEnv(); agent.Socket != "/tmp/webycp.sock" {
		t.Fatalf("AgentFromEnv().Socket = %q, want environment override", agent.Socket)
	}
}
