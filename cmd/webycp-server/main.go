package main

import (
	"context"
	"fmt"
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
	"github.com/GVALFER/WEBYCP/internal/databases"
	dnscontrol "github.com/GVALFER/WEBYCP/internal/dns"
	"github.com/GVALFER/WEBYCP/internal/ftp"
	"github.com/GVALFER/WEBYCP/internal/httpapi"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/nodes"
	"github.com/GVALFER/WEBYCP/internal/packages"
	"github.com/GVALFER/WEBYCP/internal/services"
	"github.com/GVALFER/WEBYCP/internal/signalx"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite"
	"github.com/GVALFER/WEBYCP/internal/tasks"
	"github.com/GVALFER/WEBYCP/internal/websites"
)

func main() {
	if buildinfo.Show(os.Args, os.Stdout) {
		return
	}
	args := os.Args[1:]
	migrateOnly := len(args) == 1 && args[0] == "migrate"
	checkSchema := len(args) == 1 && args[0] == "check-schema"
	adminCommand := (len(args) == 2 && args[0] == "admin" && args[1] == "init") ||
		(len(args) == 3 && args[0] == "admin" && args[1] == "reset-password")
	if len(args) > 0 && !migrateOnly && !checkSchema && !adminCommand {
		usage()
		os.Exit(2)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signalx.Context()
	defer stop()
	settings := config.ServerFromEnv()

	if checkSchema {
		if err := sqlite.CheckSchema(ctx, settings.DatabasePath); err != nil {
			logger.Error("database schema is not compatible", "error", err)
			os.Exit(1)
		}
		logger.Info("database schema is compatible")
		return
	}

	store, err := sqlite.Open(ctx, settings.DatabasePath)
	if err != nil {
		logger.Error("failed to open database", "error", err, "path", settings.DatabasePath)
		os.Exit(1)
	}
	if migrateOnly {
		if err := store.Close(); err != nil {
			logger.Error("failed to close database", "error", err)
			os.Exit(1)
		}
		logger.Info("database migrations complete", "path", settings.DatabasePath)
		return
	}
	authService := auth.NewService(store)
	if adminCommand {
		if err := runAdmin(ctx, authService, args); err != nil {
			fmt.Fprintln(os.Stderr, "WEBYCP admin:", err)
			store.Close()
			os.Exit(1)
		}
		if err := store.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "WEBYCP admin: close database:", err)
			os.Exit(1)
		}
		return
	}
	defer store.Close()

	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "localhost"
	}
	node, err := store.EnsureLocal(ctx, hostname, settings.AgentSocket)
	if err != nil {
		logger.Error("failed to register local node", "error", err)
		os.Exit(1)
	}

	agent := agentclient.New(2 * time.Minute)
	nodeService := nodes.NewService(store, agent)
	if err := nodeService.Probe(ctx, node.ID); err != nil {
		logger.Warn("initial agent probe failed", "error", err, "nodeId", node.ID)
	}
	worker := jobs.NewWorker(store, store, logger)
	packageService := packages.NewService(store)
	serviceService := services.NewService(store)
	accountService := accounts.NewService(store, store, agent, packageService, worker.Notify)
	dnsService := dnscontrol.NewService(store, accountService, store, agent, worker.Notify)
	if _, err := dnsService.EnsureLocalProvider(ctx, node.ID); err != nil {
		logger.Error("failed to register local DNS provider", "error", err)
		os.Exit(1)
	}
	websiteService := websites.NewService(store, accountService, store, agent, worker.Notify)
	databaseService := databases.NewService(store, accountService, store, agent, worker.Notify)
	taskService := tasks.NewService(store, accountService, store, agent, worker.Notify)
	ftpService := ftp.NewService(store, accountService, store, agent, worker.Notify)
	certificateService := certificates.NewService(store, websiteService, accountService, store, agent, worker.Notify)
	backupService := backups.NewService(store, accountService, websiteService, databaseService, taskService, certificateService, store, agent, worker.Notify)
	worker.Handle(jobs.KindAccountCreate, accountService.Provision)
	worker.Handle(jobs.KindAccountDelete, accountService.ProvisionAction)
	worker.Handle(jobs.KindAccountDisable, accountService.ProvisionAction)
	worker.Handle(jobs.KindAccountEnable, accountService.ProvisionAction)
	worker.Handle(jobs.KindWebsiteCreate, websiteService.Provision)
	worker.Handle(jobs.KindWebsiteDelete, websiteService.ProvisionWebsiteAction)
	worker.Handle(jobs.KindWebsiteDisable, websiteService.ProvisionWebsiteAction)
	worker.Handle(jobs.KindWebsiteEnable, websiteService.ProvisionWebsiteAction)
	worker.Handle(jobs.KindWebsiteDomainCreate, websiteService.ProvisionDomain)
	worker.Handle(jobs.KindWebsiteDomainDelete, websiteService.ProvisionDomain)
	worker.Handle(jobs.KindWebsiteDomainDisable, websiteService.ProvisionDomain)
	worker.Handle(jobs.KindWebsiteDomainEnable, websiteService.ProvisionDomain)
	worker.Handle(jobs.KindWebsiteDomainUpdate, websiteService.ProvisionDomainRename)
	worker.Handle(jobs.KindDatabaseCreate, databaseService.Provision)
	worker.Handle(jobs.KindDatabaseDelete, databaseService.Provision)
	worker.Handle(jobs.KindDatabaseUserCreate, databaseService.Provision)
	worker.Handle(jobs.KindDatabaseUserDelete, databaseService.Provision)
	worker.Handle(jobs.KindDatabaseGrantCreate, databaseService.Provision)
	worker.Handle(jobs.KindDatabaseGrantDelete, databaseService.Provision)
	worker.Handle(jobs.KindTaskSync, taskService.Sync)
	worker.Handle(jobs.KindFTPSync, ftpService.Sync)
	worker.Handle(jobs.KindCertificateIssue, certificateService.Provision)
	worker.Handle(jobs.KindCertificateRenew, certificateService.Provision)
	worker.Handle(jobs.KindBackupCreate, backupService.Create)
	worker.Handle(jobs.KindBackupRestore, backupService.RestoreJob)
	worker.Handle(jobs.KindDNSZoneCreate, dnsService.ProvisionZone)
	worker.Handle(jobs.KindDNSZoneDelete, dnsService.ProvisionZone)
	worker.Handle(jobs.KindDNSRecordSync, dnsService.ProvisionRecord)
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
			Version:      buildinfo.Version,
			SecureCookie: settings.SecureCookie, Auth: authService,
			Accounts: accountService, Packages: packageService, Services: serviceService,
			Websites: websiteService, Databases: databaseService, Tasks: taskService,
			FTP:          ftpService,
			Certificates: certificateService,
			Backups:      backupService,
			DNS:          dnsService,
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

func usage() {
	fmt.Fprintln(
		os.Stderr,
		"usage: webycp-server [--version|migrate|check-schema|admin init|admin reset-password USERNAME]",
	)
}

func runAdmin(ctx context.Context, service *auth.Service, args []string) error {
	if args[1] == "init" {
		credentials, created, err := service.InitAdmin(ctx)
		if err != nil {
			return err
		}
		if !created {
			fmt.Fprintln(os.Stdout, "Initial administrator already exists.")
			return nil
		}
		printCredentials("Initial administrator created.", credentials)
		return nil
	}
	credentials, err := service.ResetPassword(ctx, args[2])
	if err != nil {
		return err
	}
	printCredentials("Administrator password reset.", credentials)
	return nil
}

func printCredentials(message string, credentials auth.Credentials) {
	fmt.Fprintln(os.Stdout, message)
	fmt.Fprintln(os.Stdout, "Username:", credentials.Username)
	fmt.Fprintln(os.Stdout, "Temporary password:", credentials.Password)
	fmt.Fprintln(os.Stdout, "Password change required on first login.")
}
