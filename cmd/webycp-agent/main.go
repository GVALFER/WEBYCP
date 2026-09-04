package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	agentaccount "github.com/GVALFER/WEBYCP/internal/agent/account"
	backuplocal "github.com/GVALFER/WEBYCP/internal/agent/backup/local"
	"github.com/GVALFER/WEBYCP/internal/agent/capability"
	"github.com/GVALFER/WEBYCP/internal/agent/certificate/certbot"
	"github.com/GVALFER/WEBYCP/internal/agent/database/mysql"
	"github.com/GVALFER/WEBYCP/internal/agent/runtime/phpfpm"
	"github.com/GVALFER/WEBYCP/internal/agent/scheduler/crontab"
	agentserver "github.com/GVALFER/WEBYCP/internal/agent/server"
	"github.com/GVALFER/WEBYCP/internal/agent/webserver/nginx"
	agentwebsite "github.com/GVALFER/WEBYCP/internal/agent/website"
	"github.com/GVALFER/WEBYCP/internal/buildinfo"
	"github.com/GVALFER/WEBYCP/internal/config"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/signalx"
)

func main() {
	if buildinfo.Show(os.Args, os.Stdout) {
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signalx.Context()
	defer stop()

	settings := config.AgentFromEnv()
	listener, cleanup, err := agentserver.Listen(settings.Socket)
	if err != nil {
		logger.Error("failed to listen", "error", err, "socket", settings.Socket)
		os.Exit(1)
	}
	defer cleanup()

	nginxDriver := nginx.New()
	runtimeDriver := phpfpm.New()
	websiteManager := agentwebsite.New(runtimeDriver, nginxDriver)
	accountManager := agentaccount.New(agentaccount.NewLinux(), runtimeDriver)
	server := &http.Server{
		Handler: agentserver.New(agentserver.Options{
			Version: buildinfo.Version, Capabilities: capability.New(),
			Accounts: accountManager, AccountActions: accountManager,
			Websites: websiteManager, Databases: mysql.New(), Cron: crontab.New(),
			Certificates: certbot.New(nginxDriver), Logger: logger,
			Backups: backuplocal.New(),
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("agent started", "socket", settings.Socket, "version", buildinfo.Version)
	if err := httpx.Serve(ctx, server, listener); err != nil {
		logger.Error("agent stopped with an error", "error", err)
		os.Exit(1)
	}

	logger.Info("agent stopped")
}
