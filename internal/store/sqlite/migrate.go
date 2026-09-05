package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"

	"github.com/GVALFER/WEBYCP/migrations"
)

// CheckSchema verifies an existing database without applying migrations.
func CheckSchema(ctx context.Context, path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	dsn := (&url.URL{Scheme: "file", Path: absolute, RawQuery: "mode=ro"}).String()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = migrationHistory(ctx, db)
	return err
}

func migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at INTEGER NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	entries, err := migrations.Files.ReadDir(".")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	applied, err := migrationHistory(ctx, db)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || applied[entry.Name()] {
			continue
		}

		contents, err := migrations.Files.ReadFile(entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", entry.Name(), err)
		}
		if _, err := tx.ExecContext(ctx, string(contents)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO schema_migrations (name, applied_at) VALUES (?, unixepoch())", entry.Name(),
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", entry.Name(), err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}

func migrationHistory(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, "SELECT name FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("read migration history: %w", err)
	}
	defer rows.Close()
	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("read migration name: %w", err)
		}
		if _, err := migrations.Files.ReadFile(name); err != nil {
			return nil, fmt.Errorf("database migration %q is not supported by this release; use the matching release or a fresh installation", name)
		}
		applied[name] = true
	}
	return applied, rows.Err()
}
