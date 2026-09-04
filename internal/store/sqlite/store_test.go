package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/backupfmt"
	"github.com/GVALFER/WEBYCP/internal/backups"
	cronjob "github.com/GVALFER/WEBYCP/internal/cron"
	"github.com/GVALFER/WEBYCP/internal/databases"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/packages"
	"github.com/GVALFER/WEBYCP/internal/services"
	migrationfiles "github.com/GVALFER/WEBYCP/migrations"
)

func TestWebsiteMigrationConvertsExistingResources(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "webycp.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		t.Fatal(err)
	}
	if err := applyMigrationsBefore(ctx, db, "0008_websites.sql"); err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, email, name, password_hash, role, created_at, updated_at, username, timezone)
		VALUES ('user-1', 'admin@example.com', 'Admin', 'hash', 'admin', 1, 1, 'admin', 'UTC');
		INSERT INTO nodes (id, name, kind, endpoint, status, created_at, updated_at)
		VALUES ('node-1', 'Local', 'local', '/run/webycp/agent.sock', 'online', 1, 1);
		INSERT INTO accounts (id, node_id, name, system_user, status, created_at, updated_at)
		VALUES ('account-1', 'node-1', 'Customer', 'wcp_customer', 'active', 1, 1);
		INSERT INTO account_members (account_id, user_id, role, created_at)
		VALUES ('account-1', 'user-1', 'owner', 1);
		INSERT INTO domains (id, account_id, node_id, name, status, php_version, enabled, created_at, updated_at)
		VALUES ('website-1', 'account-1', 'node-1', 'example.com', 'active', '8.3', 1, 2, 3);
		INSERT INTO domain_aliases (id, domain_id, name, status, enabled, created_at, updated_at)
		VALUES ('domain-2', 'website-1', 'www.example.com', 'active', 1, 2, 3);
		INSERT INTO certificates (id, domain_id, node_id, kind, name, email, status, created_at, updated_at)
		VALUES ('certificate-1', 'website-1', 'node-1', 'domain', 'example.com', 'admin@example.com', 'active', 2, 3);
		INSERT INTO certificate_names (certificate_id, name)
		VALUES ('certificate-1', 'example.com'), ('certificate-1', 'www.example.com');
		INSERT INTO jobs (id, node_id, user_id, kind, status, payload, max_attempts, created_at)
		VALUES ('job-1', 'node-1', 'user-1', 'domain.update', 'succeeded', '{"domainId":"website-1","previousName":"old.example.com","name":"example.com"}', 1, 2);
		INSERT INTO audit_events (id, user_id, action, resource_type, resource_id, result, created_at)
		VALUES ('audit-1', 'user-1', 'alias.create', 'domain_alias', 'domain-2', 'success', 2);
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	website, err := store.Website(ctx, "website-1")
	if err != nil {
		t.Fatal(err)
	}
	if website.Name != "example.com" || website.DocumentRoot != "/home/wcp_customer/web/example.com/public_html" || website.WebDriver != "nginx" || website.RuntimeDriver != "phpfpm" || website.RuntimeVersion != "8.3" {
		t.Fatalf("migrated website = %+v", website)
	}
	primary, err := store.PrimaryDomain(ctx, website.ID)
	if err != nil || primary.Hostname != "example.com" || primary.Kind != "primary" {
		t.Fatalf("migrated primary domain = %+v, error = %v", primary, err)
	}
	alias, err := store.WebsiteDomain(ctx, "domain-2")
	if err != nil || alias.Hostname != "www.example.com" || alias.Kind != "alias" {
		t.Fatalf("migrated alias = %+v, error = %v", alias, err)
	}
	certificate, err := store.Certificate(ctx, "certificate-1")
	if err != nil || certificate.WebsiteID != website.ID || certificate.Kind != "website" || len(certificate.Names) != 2 {
		t.Fatalf("migrated certificate = %+v, error = %v", certificate, err)
	}
	job, err := store.Job(ctx, "job-1")
	if err != nil || job.Kind != jobs.KindWebsiteDomainUpdate || job.Payload != `{"websiteDomainId":"website-1","previousHostname":"old.example.com","hostname":"example.com"}` {
		t.Fatalf("migrated job = %+v, error = %v", job, err)
	}
	var action, resourceType string
	if err := store.db.QueryRowContext(ctx, "SELECT action, resource_type FROM audit_events WHERE id = 'audit-1'").Scan(&action, &resourceType); err != nil {
		t.Fatal(err)
	}
	if action != "website_domain.create" || resourceType != "website_domain" {
		t.Fatalf("migrated audit event = %s, %s", action, resourceType)
	}
	overview, err := store.AccountOverview(ctx, "account-1")
	if err != nil {
		t.Fatal(err)
	}
	if overview.Package.ID != packages.DefaultID || overview.Usage.Websites != 1 || overview.Usage.Domains != 1 || overview.Usage.Aliases != 1 {
		t.Fatalf("migrated Account Package = %+v", overview)
	}
	for _, table := range []string{"domains", "domain_aliases"} {
		var count int
		if err := store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("legacy table %s still exists", table)
		}
	}
}

func TestAdminCredentialMigrationPreservesExistingLogin(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "webycp.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
		CREATE TABLE schema_migrations (name TEXT PRIMARY KEY, applied_at INTEGER NOT NULL);
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL COLLATE NOCASE UNIQUE,
			name TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL CHECK (role IN ('admin', 'user')),
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);
		CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			csrf_token TEXT NOT NULL,
			expires_at INTEGER NOT NULL,
			created_at INTEGER NOT NULL
		);
	`)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"0001_control_plane.sql", "0002_domains.sql", "0003_domain_lifecycle.sql",
		"0004_domain_names.sql", "0005_v1_resources.sql",
		"0008_websites.sql", "0009_packages.sql", "0010_service_capabilities.sql",
	} {
		if _, err := db.ExecContext(ctx, "INSERT INTO schema_migrations VALUES (?, unixepoch())", name); err != nil {
			t.Fatal(err)
		}
	}
	passwordHash, err := auth.HashPassword("existing secure password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO users VALUES (?, ?, ?, ?, 'admin', unixepoch(), unixepoch())
	`, "0123456789abcdef0123456789abcdef", "admin@example.com", "Admin", passwordHash); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := auth.NewService(store)
	session, err := service.Login(ctx, "admin", "existing secure password")
	if err != nil {
		t.Fatal(err)
	}
	if session.User.Username != "admin" || session.User.Timezone != "UTC" || session.User.MustChangePassword {
		t.Fatalf("migrated user = %+v", session.User)
	}
}

func TestOpenRunsMigrationsAndPersistsAdmin(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "webycp.db")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	created, err := store.InitAdmin(ctx, auth.NewUser{
		ID:           "user-1",
		Username:     "admin",
		Email:        "admin@example.com",
		Name:         "Admin",
		PasswordHash: "hash",
		Role:         "admin",
		CreatedAt:    now,
	})
	if err != nil || !created {
		t.Fatalf("create admin: created=%v, error=%v", created, err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	count, err := reopened.UserCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one user, got %d", count)
	}
}

func TestNodeCapabilitiesAndServiceSettings(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "webycp.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	node, err := store.EnsureLocal(ctx, "test", "/tmp/agent.sock")
	if err != nil {
		t.Fatal(err)
	}
	observed := time.Now().UTC()
	capabilities := services.Capabilities{
		Webservers: []services.Capability{{Driver: services.Nginx, Status: services.Healthy}},
		Runtimes:   []services.Capability{{Driver: services.PHPFPM, Version: services.PHP83, Status: services.Healthy}},
	}
	if err := store.UpdateProbe(ctx, node.ID, "online", &observed, &capabilities); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateProbe(ctx, node.ID, "offline", nil, nil); err != nil {
		t.Fatal(err)
	}
	stored, err := store.Node(ctx, node.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != "offline" || stored.Capabilities == nil ||
		stored.Capabilities.Webservers[0].Driver != services.Nginx ||
		stored.CapabilitiesAt == nil || stored.LastSeenAt == nil {
		t.Fatalf("node = %+v", stored)
	}

	settings, err := store.ServiceSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if settings.Defaults.WebDriver != services.Nginx ||
		settings.Defaults.DatabaseDriver != services.MySQL {
		t.Fatalf("settings = %+v", settings)
	}
	settings.Defaults = services.Defaults{
		WebDriver: services.Nginx, RuntimeDriver: services.PHPFPM,
		RuntimeVersion: services.PHP83, DatabaseDriver: services.MySQL,
		SchedulerDriver: services.Crontab, BackupDriver: services.Local,
	}
	settings.UpdatedAt = observed
	updated, err := store.UpdateServiceSettings(ctx, settings)
	if err != nil || updated.Defaults != settings.Defaults {
		t.Fatalf("updated settings = %+v, error = %v", updated, err)
	}
}

func TestJobRetryAndCompletion(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "webycp.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	now := time.Now().UTC()
	created, err := store.CreateJob(ctx, jobs.Job{
		ID:          "job-1",
		Kind:        jobs.KindNodeProbe,
		Payload:     "{}",
		MaxAttempts: 2,
		CreatedAt:   now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != "queued" {
		t.Fatalf("expected queued job, got %s", created.Status)
	}

	claimed, err := store.ClaimJob(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if claimed.Attempts != 1 {
		t.Fatalf("expected one attempt, got %d", claimed.Attempts)
	}
	if err := store.RetryJob(ctx, claimed.ID, "temporary failure"); err != nil {
		t.Fatal(err)
	}
	claimed, err = store.ClaimJob(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if claimed.Attempts != 2 {
		t.Fatalf("expected two attempts, got %d", claimed.Attempts)
	}
	if err := store.CompleteJob(ctx, claimed.ID, now); err != nil {
		t.Fatal(err)
	}
	completed, err := store.Job(ctx, claimed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != "succeeded" {
		t.Fatalf("expected succeeded job, got %s", completed.Status)
	}
}

func TestInterruptedJobDoesNotConsumeAnAttempt(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "webycp.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	now := time.Now().UTC()
	_, err = store.CreateJob(ctx, jobs.Job{
		ID: "job-1", Kind: jobs.KindNodeProbe, Payload: "{}",
		MaxAttempts: 1, CreatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := store.ClaimJob(ctx, now)
	if err != nil || claimed.Attempts != 1 {
		t.Fatalf("claim = %+v, error = %v", claimed, err)
	}
	if err := store.RecoverJobs(ctx, now); err != nil {
		t.Fatal(err)
	}
	recovered, err := store.ClaimJob(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Attempts != 1 {
		t.Fatalf("recovered attempts = %d, want 1", recovered.Attempts)
	}
}

func TestV1ResourceStoresCreateAtomically(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "webycp.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	if created, err := store.InitAdmin(ctx, auth.NewUser{ID: "user-1", Username: "admin", Email: "admin@example.com", Name: "Admin", PasswordHash: "hash", Role: "admin", CreatedAt: now}); err != nil || !created {
		t.Fatalf("create admin: created=%v, error=%v", created, err)
	}
	node, err := store.EnsureLocal(ctx, "test", "/tmp/test-agent.sock")
	if err != nil {
		t.Fatal(err)
	}
	accountID := "0123456789abcdef0123456789abcdef"
	account, _, err := store.CreateProvision(ctx, accounts.Account{ID: accountID, NodeID: node.ID, Name: "Test", SystemUser: "wcp_0123456789ab", Status: "pending", Enabled: true, CreatedAt: now, UpdatedAt: now}, "user-1", packages.DefaultID, testJob("job-account", node.ID, "user-1", jobs.KindAccountCreate, now))
	if err != nil {
		t.Fatal(err)
	}
	if !account.Enabled {
		t.Fatal("new account should be enabled")
	}
	if err := store.UpdateStatus(ctx, account.ID, "active"); err != nil {
		t.Fatal(err)
	}
	database, job, err := store.CreateDatabase(ctx, databases.Database{ID: "abcdef0123456789abcdef0123456789", AccountID: account.ID, NodeID: node.ID, Name: "app", SystemName: "wcp_01234567_app", Driver: services.MySQL, Status: "pending", CreatedAt: now, UpdatedAt: now}, testJob("job-database", node.ID, "user-1", jobs.KindDatabaseCreate, now))
	if err != nil || database.Name != "app" || job.Status != "queued" {
		t.Fatalf("database = %+v, job = %+v, error = %v", database, job, err)
	}
	cron, _, err := store.CreateCronJob(ctx, cronjob.CronJob{ID: "fedcba9876543210fedcba9876543210", AccountID: account.ID, NodeID: node.ID, Name: "Hourly", Schedule: "0 * * * *", Command: "php task.php", SchedulerDriver: services.Crontab, Enabled: true, Status: "pending", CreatedAt: now, UpdatedAt: now}, testJob("job-cron", node.ID, "user-1", jobs.KindCronSync, now))
	if err != nil || !cron.Enabled {
		t.Fatalf("cron = %+v, error = %v", cron, err)
	}
	plan, err := store.CreateBackupPlan(ctx, backups.Plan{ID: "11111111111111111111111111111111", AccountID: account.ID, NodeID: node.ID, Name: "Daily", Schedule: "0 3 * * *", RetentionCount: 7, StorageDriver: services.Local, IncludeFiles: true, IncludeDatabases: true, Enabled: true, CreatedAt: now, UpdatedAt: now})
	if err != nil || plan.RetentionCount != 7 {
		t.Fatalf("plan = %+v, error = %v", plan, err)
	}
	metadata := backupfmt.Metadata{
		Version: backupfmt.Version, AccountID: account.ID,
		Websites: []backupfmt.Website{{
			ID: "22222222222222222222222222222222", NodeID: node.ID, Name: "Restored",
			Kind: "php", DocumentRoot: "/home/wcp_0123456789ab/web/22222222222222222222222222222222/public_html",
			WebDriver: "nginx", RuntimeDriver: "phpfpm", RuntimeVersion: "8.3", Enabled: true,
			Domains: []backupfmt.WebsiteDomain{
				{ID: "66666666666666666666666666666666", Hostname: "restored.example.com", Kind: "primary", Enabled: true},
				{ID: "33333333333333333333333333333333", Hostname: "www.restored.example.com", Kind: "alias", Enabled: true},
			},
		}},
		Databases: []backupfmt.Database{{
			ID: "44444444444444444444444444444444", NodeID: node.ID,
			Name: "restored", SystemName: "wcp_01234567_restored", Driver: services.MySQL,
		}},
		CronJobs: []backupfmt.CronJob{{
			ID: "55555555555555555555555555555555", NodeID: node.ID,
			Name: "Restored", Schedule: "15 * * * *", Command: "php task.php",
			SchedulerDriver: services.Crontab, Enabled: true,
		}},
	}
	if err := store.RestoreMetadata(ctx, metadata); err != nil {
		t.Fatal(err)
	}
	if website, err := store.Website(ctx, metadata.Websites[0].ID); err != nil || website.Name != "Restored" {
		t.Fatalf("restored website = %+v, error = %v", website, err)
	}
	if domain, err := store.WebsiteDomain(ctx, metadata.Websites[0].Domains[1].ID); err != nil || domain.Hostname != "www.restored.example.com" {
		t.Fatalf("restored domain = %+v, error = %v", domain, err)
	}
	if database, err := store.Database(ctx, metadata.Databases[0].ID); err != nil || database.Status != "active" {
		t.Fatalf("restored database = %+v, error = %v", database, err)
	}
	if cron, err := store.CronJob(ctx, metadata.CronJobs[0].ID); err != nil || cron.Status != "active" {
		t.Fatalf("restored cron job = %+v, error = %v", cron, err)
	}
	metadata.Websites[0].NodeID = "ffffffffffffffffffffffffffffffff"
	if err := store.RestoreMetadata(ctx, metadata); err == nil {
		t.Fatal("expected cross-node restore metadata to be rejected")
	}
}

func testJob(id, nodeID, userID, kind string, now time.Time) jobs.Job {
	return jobs.Job{ID: id, NodeID: nodeID, UserID: userID, Kind: kind, Payload: "{}", MaxAttempts: 1, CreatedAt: now}
}

func applyMigrationsBefore(ctx context.Context, db *sql.DB, stop string) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at INTEGER NOT NULL
		)
	`); err != nil {
		return err
	}
	entries, err := migrationfiles.Files.ReadDir(".")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() >= stop {
			continue
		}
		contents, err := migrationfiles.Files.ReadFile(entry.Name())
		if err != nil {
			return err
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(contents)); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations VALUES (?, unixepoch())", entry.Name()); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
