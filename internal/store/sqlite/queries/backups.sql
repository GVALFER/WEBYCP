-- name: ListBackupPlans :many
SELECT backup_plans.* FROM backup_plans
JOIN account_members ON account_members.account_id = backup_plans.account_id
WHERE ? OR account_members.user_id = ?
GROUP BY backup_plans.id
ORDER BY backup_plans.created_at ASC;

-- name: CountBackupPlans :one
SELECT COUNT(DISTINCT backup_plans.id) FROM backup_plans
JOIN account_members ON account_members.account_id = backup_plans.account_id
WHERE sqlc.arg(is_admin) OR account_members.user_id = sqlc.arg(user_id);

-- name: ListBackupPlansPage :many
SELECT backup_plans.* FROM backup_plans
JOIN account_members ON account_members.account_id = backup_plans.account_id
WHERE sqlc.arg(is_admin) OR account_members.user_id = sqlc.arg(user_id)
GROUP BY backup_plans.id
ORDER BY backup_plans.created_at ASC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: GetBackupPlan :one
SELECT * FROM backup_plans WHERE id = ? LIMIT 1;

-- name: CreateBackupPlan :one
INSERT INTO backup_plans (
    id, account_id, node_id, name, schedule, retention_count,
    include_files, include_databases, enabled, next_run_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateBackupPlan :one
UPDATE backup_plans SET
    name = ?, schedule = ?, retention_count = ?, include_files = ?,
    include_databases = ?, enabled = ?, next_run_at = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteBackupPlan :exec
DELETE FROM backup_plans WHERE id = ?;

-- name: BackupRunPending :one
SELECT EXISTS(SELECT 1 FROM backup_runs WHERE plan_id = ? AND status IN ('queued', 'running'));

-- name: ListDueBackupPlans :many
SELECT * FROM backup_plans
WHERE enabled = 1 AND schedule <> '' AND next_run_at IS NOT NULL AND next_run_at <= ?
ORDER BY next_run_at;

-- name: MarkBackupPlanRun :exec
UPDATE backup_plans SET last_run_at = ?, next_run_at = ?, updated_at = ? WHERE id = ?;

-- name: CreateBackupRun :one
INSERT INTO backup_runs (
    id, plan_id, account_id, node_id, status, created_at
) VALUES (?, ?, ?, ?, 'queued', ?)
RETURNING *;

-- name: UpdateBackupRun :exec
UPDATE backup_runs SET status = ?, error = ?,
    started_at = COALESCE(?, started_at), finished_at = COALESCE(?, finished_at)
WHERE id = ?;

-- name: GetBackupRun :one
SELECT * FROM backup_runs WHERE id = ? LIMIT 1;

-- name: ListBackupRuns :many
SELECT backup_runs.* FROM backup_runs
JOIN account_members ON account_members.account_id = backup_runs.account_id
WHERE ? OR account_members.user_id = ?
GROUP BY backup_runs.id
ORDER BY backup_runs.created_at DESC;

-- name: CountBackupRuns :one
SELECT COUNT(DISTINCT backup_runs.id) FROM backup_runs
JOIN account_members ON account_members.account_id = backup_runs.account_id
WHERE sqlc.arg(is_admin) OR account_members.user_id = sqlc.arg(user_id);

-- name: ListBackupRunsPage :many
SELECT backup_runs.* FROM backup_runs
JOIN account_members ON account_members.account_id = backup_runs.account_id
WHERE sqlc.arg(is_admin) OR account_members.user_id = sqlc.arg(user_id)
GROUP BY backup_runs.id
ORDER BY backup_runs.created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CreateBackupArtifact :one
INSERT INTO backup_artifacts (
    id, run_id, account_id, node_id, path, checksum, size_bytes, manifest, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetBackupArtifact :one
SELECT * FROM backup_artifacts WHERE id = ? LIMIT 1;

-- name: ListBackupArtifacts :many
SELECT backup_artifacts.* FROM backup_artifacts
JOIN account_members ON account_members.account_id = backup_artifacts.account_id
WHERE ? OR account_members.user_id = ?
GROUP BY backup_artifacts.id
ORDER BY backup_artifacts.created_at DESC;

-- name: CountBackupArtifacts :one
SELECT COUNT(DISTINCT backup_artifacts.id) FROM backup_artifacts
JOIN account_members ON account_members.account_id = backup_artifacts.account_id
WHERE sqlc.arg(is_admin) OR account_members.user_id = sqlc.arg(user_id);

-- name: ListBackupArtifactsPage :many
SELECT backup_artifacts.* FROM backup_artifacts
JOIN account_members ON account_members.account_id = backup_artifacts.account_id
WHERE sqlc.arg(is_admin) OR account_members.user_id = sqlc.arg(user_id)
GROUP BY backup_artifacts.id
ORDER BY backup_artifacts.created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: ListExpiredBackupArtifacts :many
SELECT backup_artifacts.* FROM backup_artifacts
JOIN backup_runs ON backup_runs.id = backup_artifacts.run_id
WHERE backup_runs.plan_id = ?
ORDER BY backup_artifacts.created_at DESC
LIMIT -1 OFFSET ?;

-- name: DeleteBackupArtifact :exec
DELETE FROM backup_artifacts WHERE id = ?;

-- name: UpsertRestoredWebsite :exec
INSERT INTO websites (
    id, account_id, node_id, name, kind, document_root, web_driver,
    runtime_driver, runtime_version, status, enabled, created_at, updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CASE WHEN ? = 1 THEN 'active' ELSE 'disabled' END, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET name = excluded.name, kind = excluded.kind,
    document_root = excluded.document_root, web_driver = excluded.web_driver,
    runtime_driver = excluded.runtime_driver, runtime_version = excluded.runtime_version,
    status = excluded.status, enabled = excluded.enabled, updated_at = excluded.updated_at;

-- name: UpsertRestoredWebsiteDomain :exec
INSERT INTO website_domains (
    id, website_id, hostname, kind, status, enabled, created_at, updated_at
)
VALUES (?, ?, ?, ?, CASE WHEN ? = 1 THEN 'active' ELSE 'disabled' END, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET hostname = excluded.hostname, kind = excluded.kind,
    status = excluded.status, enabled = excluded.enabled, updated_at = excluded.updated_at;

-- name: UpsertRestoredDatabase :exec
INSERT INTO databases (id, account_id, node_id, name, system_name, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, 'active', ?, ?)
ON CONFLICT(id) DO UPDATE SET name = excluded.name, system_name = excluded.system_name,
    status = 'active', updated_at = excluded.updated_at;

-- name: UpsertRestoredCronJob :exec
INSERT INTO cron_jobs (id, account_id, node_id, name, schedule, command, enabled, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, CASE WHEN ? = 1 THEN 'active' ELSE 'disabled' END, ?, ?)
ON CONFLICT(id) DO UPDATE SET name = excluded.name, schedule = excluded.schedule,
    command = excluded.command, enabled = excluded.enabled,
    status = CASE WHEN excluded.enabled = 1 THEN 'active' ELSE 'disabled' END,
    updated_at = excluded.updated_at;
