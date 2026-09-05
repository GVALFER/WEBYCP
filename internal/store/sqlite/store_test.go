package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/backupfmt"
	"github.com/GVALFER/WEBYCP/internal/backups"
	"github.com/GVALFER/WEBYCP/internal/databases"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/packages"
	"github.com/GVALFER/WEBYCP/internal/services"
	"github.com/GVALFER/WEBYCP/internal/tasks"
)

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
	task, _, err := store.CreateScheduledTask(ctx, tasks.ScheduledTask{ID: "fedcba9876543210fedcba9876543210", AccountID: account.ID, NodeID: node.ID, Name: "Hourly", Schedule: "0 * * * *", Command: "php task.php", SchedulerDriver: services.Crontab, Kind: "command", Enabled: true, Status: "pending", CreatedAt: now, UpdatedAt: now}, testJob("job-task", node.ID, "user-1", jobs.KindTaskSync, now))
	if err != nil || !task.Enabled {
		t.Fatalf("task = %+v, error = %v", task, err)
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
		ScheduledTasks: []backupfmt.ScheduledTask{{
			ID: "55555555555555555555555555555555", NodeID: node.ID,
			Name: "Restored", Schedule: "15 * * * *", Command: "php task.php",
			SchedulerDriver: services.Crontab, Kind: "command", Enabled: true,
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
	if task, err := store.ScheduledTask(ctx, metadata.ScheduledTasks[0].ID); err != nil || task.Status != "active" {
		t.Fatalf("restored scheduled task = %+v, error = %v", task, err)
	}
	metadata.Websites[0].NodeID = "ffffffffffffffffffffffffffffffff"
	if err := store.RestoreMetadata(ctx, metadata); err == nil {
		t.Fatal("expected cross-node restore metadata to be rejected")
	}
}

func testJob(id, nodeID, userID, kind string, now time.Time) jobs.Job {
	return jobs.Job{ID: id, NodeID: nodeID, UserID: userID, Kind: kind, Payload: "{}", MaxAttempts: 1, CreatedAt: now}
}
