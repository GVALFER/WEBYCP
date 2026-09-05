package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GVALFER/WEBYCP/internal/packages"
)

func TestBaseline(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "webycp.db")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := CheckSchema(ctx, path); err != nil {
		t.Fatal(err)
	}
	var name string
	if err := store.db.QueryRowContext(ctx, "SELECT name FROM schema_migrations").Scan(&name); err != nil || name != "0001_initial.sql" {
		t.Fatalf("migration = %q, error = %v", name, err)
	}
	for _, table := range []string{"users", "sessions", "accounts", "nodes", "websites", "website_domains", "certificates", "databases", "database_users", "scheduled_tasks", "backup_plans", "backup_runs", "backup_artifacts", "dns_zones", "dns_records", "jobs", "audit_events"} {
		var count int
		if err := store.db.QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("development data in %s: count = %d, error = %v", table, count, err)
		}
	}
	var invalid int
	if err := store.db.QueryRowContext(ctx, `
		SELECT count(*) FROM sqlite_master AS m, pragma_foreign_key_list(m.name) AS f
		WHERE m.type = 'table' AND NOT EXISTS (
			SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = f."table"
		)
	`).Scan(&invalid); err != nil || invalid != 0 {
		t.Fatalf("invalid foreign key targets = %d, error = %v", invalid, err)
	}
	if err := store.db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE name IN ('domains', 'domain_aliases', 'cron_jobs')").Scan(&invalid); err != nil || invalid != 0 {
		t.Fatalf("obsolete tables = %d, error = %v", invalid, err)
	}
	var id string
	var retention, created int64
	if err := store.db.QueryRowContext(ctx, "SELECT id, max_backup_retention, created_at FROM packages").Scan(&id, &retention, &created); err != nil || id != packages.DefaultID || retention != 7 || created < 1000000000000 {
		t.Fatalf("default Package = %s, %d, %d; error = %v", id, retention, created, err)
	}
}

func TestRejectUnknownMigration(t *testing.T) {
	for _, name := range []string{"0001_control_plane.sql", "0099_future.sql"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			path := filepath.Join(t.TempDir(), "existing.db")
			db, err := sql.Open("sqlite", path)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if _, err := db.ExecContext(ctx, `
				CREATE TABLE schema_migrations (name TEXT PRIMARY KEY, applied_at INTEGER NOT NULL);
				CREATE TABLE users (id TEXT PRIMARY KEY);
				INSERT INTO users VALUES ('preserve-me');
				INSERT INTO schema_migrations VALUES (?, 1);
			`, name); err != nil {
				t.Fatal(err)
			}
			if err := CheckSchema(ctx, path); err == nil || !strings.Contains(err.Error(), "not supported by this release") {
				t.Fatalf("preflight error = %v", err)
			}
			store, err := Open(ctx, path)
			if store != nil {
				store.Close()
			}
			if err == nil || !strings.Contains(err.Error(), "not supported by this release") {
				t.Fatalf("open error = %v", err)
			}
			var id string
			if err := db.QueryRowContext(ctx, "SELECT id FROM users").Scan(&id); err != nil || id != "preserve-me" {
				t.Fatalf("existing data changed: %q, %v", id, err)
			}
			var count int
			if err := db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type = 'table'").Scan(&count); err != nil || count != 2 {
				t.Fatalf("existing schema changed: %d tables, %v", count, err)
			}
		})
	}
}

func TestCheckSchemaMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.db")
	if err := CheckSchema(context.Background(), path); err == nil {
		t.Fatal("missing database accepted")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("read-only preflight created a database: %v", err)
	}
}
