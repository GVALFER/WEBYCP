package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	agentclient "github.com/GVALFER/WEBYCP/internal/agent/client"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/backups"
	"github.com/GVALFER/WEBYCP/internal/buildinfo"
	"github.com/GVALFER/WEBYCP/internal/certificates"
	"github.com/GVALFER/WEBYCP/internal/config"
	cronjob "github.com/GVALFER/WEBYCP/internal/cron"
	"github.com/GVALFER/WEBYCP/internal/databases"
	"github.com/GVALFER/WEBYCP/internal/domains"
	"github.com/GVALFER/WEBYCP/internal/httpapi"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/nodes"
	"github.com/GVALFER/WEBYCP/internal/signalx"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite"
)

func main() {
	if buildinfo.Show(os.Args, os.Stdout) {
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signalx.Context()
	defer stop()
	settings := config.ServerFromEnv()

	store, err := sqlite.Open(ctx, settings.DatabasePath)
	if err != nil {
		logger.Error("failed to open database", "error", err, "path", settings.DatabasePath)
		os.Exit(1)
	}
	defer store.Close()

	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "localhost"
	}
	if _, err := store.EnsureLocal(ctx, hostname, settings.AgentSocket); err != nil {
		logger.Error("failed to register local node", "error", err)
		os.Exit(1)
	}

	agent := agentclient.New(5 * time.Second)
	nodeService := nodes.NewService(store, agent)
	worker := jobs.NewWorker(store, store, logger)
	accountService := accounts.NewService(store, store, agent, worker.Notify)
	domainService := domains.NewService(store, accountService, store, agent, worker.Notify)
	databaseService := databases.NewService(store, accountService, store, agent, worker.Notify)
	cronService := cronjob.NewService(store, accountService, store, agent, worker.Notify)
	certificateService := certificates.NewService(store, domainService, accountService, store, agent, worker.Notify)
	backupService := backups.NewService(store, accountService, domainService, databaseService, cronService, store, agent, worker.Notify)
	worker.Handle(jobs.KindAccountCreate, accountService.Provision)
	worker.Handle(jobs.KindAccountDelete, accountService.ProvisionAction)
	worker.Handle(jobs.KindAccountDisable, accountService.ProvisionAction)
	worker.Handle(jobs.KindAccountEnable, accountService.ProvisionAction)
	worker.Handle(jobs.KindAliasCreate, domainService.ProvisionAlias)
	worker.Handle(jobs.KindAliasDelete, domainService.ProvisionAliasAction)
	worker.Handle(jobs.KindAliasDisable, domainService.ProvisionAliasAction)
	worker.Handle(jobs.KindAliasEnable, domainService.ProvisionAliasAction)
	worker.Handle(jobs.KindAliasUpdate, domainService.ProvisionAliasRename)
	worker.Handle(jobs.KindDomainCreate, domainService.Provision)
	worker.Handle(jobs.KindDomainDelete, domainService.ProvisionDomainAction)
	worker.Handle(jobs.KindDomainDisable, domainService.ProvisionDomainAction)
	worker.Handle(jobs.KindDomainEnable, domainService.ProvisionDomainAction)
	worker.Handle(jobs.KindDomainUpdate, domainService.ProvisionDomainRename)
	worker.Handle(jobs.KindDatabaseCreate, databaseService.Provision)
	worker.Handle(jobs.KindDatabaseDelete, databaseService.Provision)
	worker.Handle(jobs.KindDatabaseUserCreate, databaseService.Provision)
	worker.Handle(jobs.KindDatabaseUserDelete, databaseService.Provision)
	worker.Handle(jobs.KindDatabaseGrantCreate, databaseService.Provision)
	worker.Handle(jobs.KindDatabaseGrantDelete, databaseService.Provision)
	worker.Handle(jobs.KindCronSync, cronService.Sync)
	worker.Handle(jobs.KindCertificateIssue, certificateService.Provision)
	worker.Handle(jobs.KindCertificateRenew, certificateService.Provision)
	worker.Handle(jobs.KindBackupCreate, backupService.Create)
	worker.Handle(jobs.KindBackupRestore, backupService.RestoreJob)
	worker.Handle(jobs.KindNodeProbe, func(ctx context.Context, job jobs.Job) error {
		return nodeService.Probe(ctx, job.NodeID)
	})
	jobService := jobs.NewService(store, worker.Notify)
	go func() {
		if err := worker.Run(ctx); err != nil {
			logger.Error("job worker stopped", "error", err)
		}
	}()
	go func() {
		if err := backupService.QueueDue(ctx); err != nil {
			logger.Error("failed to queue scheduled backups", "error", err)
		}
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := backupService.QueueDue(ctx); err != nil {
					logger.Error("failed to queue scheduled backups", "error", err)
				}
			}
		}
	}()
	go func() {
		if err := certificateService.QueueDue(ctx); err != nil {
			logger.Error("failed to queue certificate renewals", "error", err)
		}
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := certificateService.QueueDue(ctx); err != nil {
					logger.Error("failed to queue certificate renewals", "error", err)
				}
			}
		}
	}()

	listener, err := net.Listen("tcp", settings.Addr)
	if err != nil {
		logger.Error("failed to listen", "error", err, "address", settings.Addr)
		os.Exit(1)
	}

	server := &http.Server{
		Handler: httpapi.New(httpapi.Options{
			Version: buildinfo.Version, WebDir: settings.WebDir,
			SecureCookie: settings.SecureCookie, Auth: auth.NewService(store),
			Accounts: accountService, Domains: domainService, Databases: databaseService, Cron: cronService,
			Certificates: certificateService,
			Backups:      backupService,
			Nodes:        nodeService, Jobs: jobService,
			Audit: store, Logger: logger,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("server started", "address", settings.Addr, "version", buildinfo.Version)
	if err := httpx.Serve(ctx, server, listener); err != nil {
		logger.Error("server stopped with an error", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}
