package config

import "github.com/GVALFER/WEBYCP/internal/envx"

const (
	defaultServerAddr   = "127.0.0.1:8080"
	defaultDatabasePath = "/var/lib/webycp/server/webycp.db"
	defaultAgentSocket  = "/run/webycp/agent.sock"
)

type Server struct {
	Addr         string
	DatabasePath string
	AgentSocket  string
	SecureCookie bool
}

func ServerFromEnv() Server {
	return Server{
		Addr:         envx.String("WEBYCP_SERVER_ADDR", defaultServerAddr),
		DatabasePath: envx.String("WEBYCP_DATABASE_PATH", defaultDatabasePath),
		AgentSocket:  envx.String("WEBYCP_AGENT_SOCKET", defaultAgentSocket),
		SecureCookie: envx.Bool("WEBYCP_SECURE_COOKIE", true),
	}
}

type Agent struct {
	Socket string
}

func AgentFromEnv() Agent {
	return Agent{Socket: envx.String("WEBYCP_AGENT_SOCKET", defaultAgentSocket)}
}
