-- name: ListCronJobs :many
SELECT cron_jobs.* FROM cron_jobs
JOIN account_members ON account_members.account_id = cron_jobs.account_id
WHERE ? OR account_members.user_id = ?
GROUP BY cron_jobs.id
ORDER BY cron_jobs.created_at ASC;

-- name: CountCronJobs :one
SELECT COUNT(DISTINCT cron_jobs.id) FROM cron_jobs
JOIN account_members ON account_members.account_id = cron_jobs.account_id
WHERE sqlc.arg(is_admin) OR account_members.user_id = sqlc.arg(user_id);

-- name: ListCronJobsPage :many
SELECT cron_jobs.* FROM cron_jobs
JOIN account_members ON account_members.account_id = cron_jobs.account_id
WHERE sqlc.arg(is_admin) OR account_members.user_id = sqlc.arg(user_id)
GROUP BY cron_jobs.id
ORDER BY cron_jobs.created_at ASC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: GetCronJob :one
SELECT * FROM cron_jobs WHERE id = ? LIMIT 1;

-- name: ListAccountCronJobs :many
SELECT * FROM cron_jobs WHERE account_id = ? ORDER BY created_at ASC;

-- name: CreateCronJob :one
INSERT INTO cron_jobs (
    id, account_id, node_id, name, schedule, command, scheduler_driver, enabled, status,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?)
RETURNING *;

-- name: UpdateCronJob :one
UPDATE cron_jobs
SET name = ?, schedule = ?, command = ?, scheduler_driver = ?, enabled = ?, status = 'pending', updated_at = ?
WHERE id = ?
RETURNING *;

-- name: UpdateCronJobStatus :exec
UPDATE cron_jobs SET status = ?, updated_at = ? WHERE id = ?;

-- name: UpdateAccountCronStatuses :exec
UPDATE cron_jobs
SET status = CASE WHEN enabled = 1 THEN ? ELSE ? END, updated_at = ?
WHERE account_id = ?;

-- name: DeleteCronJob :exec
DELETE FROM cron_jobs WHERE id = ?;
