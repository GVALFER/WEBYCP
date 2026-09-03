package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	agentaccount "github.com/GVALFER/WEBYCP/internal/agent/account"
	backuplocal "github.com/GVALFER/WEBYCP/internal/agent/backup/local"
	"github.com/GVALFER/WEBYCP/internal/agent/certificate/certbot"
	"github.com/GVALFER/WEBYCP/internal/agent/database/mysql"
	agentdomain "github.com/GVALFER/WEBYCP/internal/agent/domain"
	"github.com/GVALFER/WEBYCP/internal/agent/runtime/phpfpm"
	"github.com/GVALFER/WEBYCP/internal/agent/scheduler/crontab"
	agentserver "github.com/GVALFER/WEBYCP/internal/agent/server"
	"github.com/GVALFER/WEBYCP/internal/agent/webserver/nginx"
	"github.com/GVALFER/WEBYCP/internal/buildinfo"
	"github.com/GVALFER/WEBYCP/internal/envx"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/signalx"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signalx.Context()
	defer stop()

	socket := envx.String("WEBYCP_AGENT_SOCKET", "/run/webycp/agent.sock")
	listener, cleanup, err := agentserver.Listen(socket)
	if err != nil {
		logger.Error("failed to listen", "error", err, "socket", socket)
		os.Exit(1)
	}
	defer cleanup()

	nginxDriver := nginx.New()
	domainManager := agentdomain.New(phpfpm.New(), nginxDriver)
	accountManager := agentaccount.NewLinux()
	server := &http.Server{
		Handler: agentserver.New(agentserver.Options{
			Version: buildinfo.Version, Accounts: accountManager, AccountActions: accountManager,
			Domains: domainManager, Databases: mysql.New(), Cron: crontab.New(),
			Certificates: certbot.New(nginxDriver), Logger: logger,
			Backups: backuplocal.New(),
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("agent started", "socket", socket, "version", buildinfo.Version)
	if err := httpx.Serve(ctx, server, listener); err != nil {
		logger.Error("agent stopped with an error", "error", err)
		os.Exit(1)
	}

	logger.Info("agent stopped")
}
