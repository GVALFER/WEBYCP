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
)

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
	account, _, err := store.CreateProvision(ctx, accounts.Account{ID: accountID, NodeID: node.ID, Name: "Test", SystemUser: "wcp_0123456789ab", Status: "pending", Enabled: true, CreatedAt: now, UpdatedAt: now}, "user-1", testJob("job-account", node.ID, "user-1", jobs.KindAccountCreate, now))
	if err != nil {
		t.Fatal(err)
	}
	if !account.Enabled {
		t.Fatal("new account should be enabled")
	}
	if err := store.UpdateStatus(ctx, account.ID, "active"); err != nil {
		t.Fatal(err)
	}
	database, job, err := store.CreateDatabase(ctx, databases.Database{ID: "abcdef0123456789abcdef0123456789", AccountID: account.ID, NodeID: node.ID, Name: "app", SystemName: "wcp_01234567_app", Status: "pending", CreatedAt: now, UpdatedAt: now}, testJob("job-database", node.ID, "user-1", jobs.KindDatabaseCreate, now))
	if err != nil || database.Name != "app" || job.Status != "queued" {
		t.Fatalf("database = %+v, job = %+v, error = %v", database, job, err)
	}
	cron, _, err := store.CreateCronJob(ctx, cronjob.CronJob{ID: "fedcba9876543210fedcba9876543210", AccountID: account.ID, NodeID: node.ID, Name: "Hourly", Schedule: "0 * * * *", Command: "php task.php", Enabled: true, Status: "pending", CreatedAt: now, UpdatedAt: now}, testJob("job-cron", node.ID, "user-1", jobs.KindCronSync, now))
	if err != nil || !cron.Enabled {
		t.Fatalf("cron = %+v, error = %v", cron, err)
	}
	plan, err := store.CreateBackupPlan(ctx, backups.Plan{ID: "11111111111111111111111111111111", AccountID: account.ID, NodeID: node.ID, Name: "Daily", Schedule: "0 3 * * *", RetentionCount: 7, IncludeFiles: true, IncludeDatabases: true, Enabled: true, CreatedAt: now, UpdatedAt: now})
	if err != nil || plan.RetentionCount != 7 {
		t.Fatalf("plan = %+v, error = %v", plan, err)
	}
	metadata := backupfmt.Metadata{
		Version: backupfmt.Version, AccountID: account.ID,
		Domains: []backupfmt.Domain{{
			ID: "22222222222222222222222222222222", NodeID: node.ID,
			Name: "restored.example.com", PHPVersion: "8.3", Enabled: true,
			Aliases: []backupfmt.Alias{{ID: "33333333333333333333333333333333", Name: "www.restored.example.com", Enabled: true}},
		}},
		Databases: []backupfmt.Database{{
			ID: "44444444444444444444444444444444", NodeID: node.ID,
			Name: "restored", SystemName: "wcp_01234567_restored",
		}},
		CronJobs: []backupfmt.CronJob{{
			ID: "55555555555555555555555555555555", NodeID: node.ID,
			Name: "Restored", Schedule: "15 * * * *", Command: "php task.php", Enabled: true,
		}},
	}
	if err := store.RestoreMetadata(ctx, metadata); err != nil {
		t.Fatal(err)
	}
	if domain, err := store.Domain(ctx, metadata.Domains[0].ID); err != nil || domain.Name != "restored.example.com" {
		t.Fatalf("restored domain = %+v, error = %v", domain, err)
	}
	if alias, err := store.Alias(ctx, metadata.Domains[0].Aliases[0].ID); err != nil || alias.Name != "www.restored.example.com" {
		t.Fatalf("restored alias = %+v, error = %v", alias, err)
	}
	if database, err := store.Database(ctx, metadata.Databases[0].ID); err != nil || database.Status != "active" {
		t.Fatalf("restored database = %+v, error = %v", database, err)
	}
	if cron, err := store.CronJob(ctx, metadata.CronJobs[0].ID); err != nil || cron.Status != "active" {
		t.Fatalf("restored cron job = %+v, error = %v", cron, err)
	}
	metadata.Domains[0].NodeID = "ffffffffffffffffffffffffffffffff"
	if err := store.RestoreMetadata(ctx, metadata); err == nil {
		t.Fatal("expected cross-node restore metadata to be rejected")
	}
}

func testJob(id, nodeID, userID, kind string, now time.Time) jobs.Job {
	return jobs.Job{ID: id, NodeID: nodeID, UserID: userID, Kind: kind, Payload: "{}", MaxAttempts: 1, CreatedAt: now}
}
