package config

import "github.com/GVALFER/WEBYCP/internal/envx"

type Server struct {
	Addr         string
	DatabasePath string
	AgentSocket  string
	WebDir       string
	SecureCookie bool
}

func ServerFromEnv() Server {
	return Server{
		Addr:         envx.String("WEBYCP_SERVER_ADDR", "127.0.0.1:8080"),
		DatabasePath: envx.String("WEBYCP_DATABASE_PATH", "/var/lib/webycp/webycp.db"),
		AgentSocket:  envx.String("WEBYCP_AGENT_SOCKET", "/run/webycp/agent.sock"),
		WebDir:       envx.String("WEBYCP_WEB_DIR", "web/dist"),
		SecureCookie: envx.Bool("WEBYCP_SECURE_COOKIE", true),
	}
}
