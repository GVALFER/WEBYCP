package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/packages"
	"github.com/GVALFER/WEBYCP/internal/tasks"
)

func TestTaskMigrationPreservesExistingSchedule(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "tasks.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := applyMigrationsBefore(ctx, db, "0012_scheduled_tasks.sql"); err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO nodes (id, name, kind, endpoint, status, created_at, updated_at)
		VALUES ('node-1', 'Local', 'local', '/run/webycp/agent.sock', 'online', 1, 1);
		INSERT INTO accounts (id, node_id, name, system_user, status, created_at, updated_at)
		VALUES ('account-1', 'node-1', 'Customer', 'wcp_customer', 'active', 1, 1);
		INSERT INTO account_packages (account_id, package_id, created_at, updated_at)
		VALUES ('account-1', ?, 1, 1);
		INSERT INTO cron_jobs (id, account_id, node_id, name, schedule, command, scheduler_driver, enabled, status, created_at, updated_at)
		VALUES ('task-1', 'account-1', 'node-1', 'Hourly', '0 * * * *', '/usr/bin/true', 'crontab', 1, 'active', 2, 3);
		INSERT INTO jobs (id, node_id, kind, status, payload, created_at)
		VALUES ('job-1', 'node-1', 'cron.sync', 'queued', '{"accountId":"account-1"}', 2);
		INSERT INTO job_steps (id, job_id, name, status, started_at)
		VALUES ('step-1', 'job-1', 'cron.sync', 'succeeded', 2);
		INSERT INTO audit_events (id, action, resource_type, resource_id, result, created_at)
		VALUES ('event-1', 'cron.create', 'cron_job', 'task-1', 'success', 2);
	`, packages.DefaultID)
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
	task, err := store.ScheduledTask(ctx, "task-1")
	if err != nil || task.Kind != tasks.Command || task.Schedule != "0 * * * *" || task.Command != "/usr/bin/true" || task.SchedulerDriver != "crontab" || !task.Enabled || task.Status != "active" || task.CreatedAt.UnixMilli() != 2 || task.UpdatedAt.UnixMilli() != 3 {
		t.Fatalf("task = %+v, error = %v", task, err)
	}
	job, err := store.Job(ctx, "job-1")
	if err != nil || job.Kind != jobs.KindTaskSync || job.Status != "queued" || job.Payload != `{"accountId":"account-1"}` {
		t.Fatalf("job = %+v, error = %v", job, err)
	}
	var step string
	if err := store.db.QueryRowContext(ctx, "SELECT name FROM job_steps WHERE id = 'step-1'").Scan(&step); err != nil || step != "task.sync" {
		t.Fatalf("step = %s, error = %v", step, err)
	}
	overview, err := store.AccountOverview(ctx, "account-1")
	if err != nil || overview.Usage.ScheduledTasks != 1 {
		t.Fatalf("usage = %+v, error = %v", overview.Usage, err)
	}
	var legacy int
	if err := store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE name = 'cron_jobs'").Scan(&legacy); err != nil || legacy != 0 {
		t.Fatalf("legacy table count = %d, error = %v", legacy, err)
	}
	var jobID sql.NullString
	if err := store.db.QueryRowContext(ctx, "SELECT job_id FROM audit_events WHERE id = 'event-1'").Scan(&jobID); err != nil || jobID.Valid {
		t.Fatalf("legacy event gained a fabricated job correlation: %+v, error = %v", jobID, err)
	}
}
