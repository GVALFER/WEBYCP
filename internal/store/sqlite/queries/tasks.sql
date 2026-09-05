-- name: ListScheduledTasks :many
SELECT scheduled_tasks.* FROM scheduled_tasks
JOIN account_members ON account_members.account_id = scheduled_tasks.account_id
WHERE ? OR account_members.user_id = ?
GROUP BY scheduled_tasks.id
ORDER BY scheduled_tasks.created_at ASC;

-- name: CountScheduledTasks :one
SELECT COUNT(DISTINCT scheduled_tasks.id) FROM scheduled_tasks
JOIN account_members ON account_members.account_id = scheduled_tasks.account_id
WHERE sqlc.arg(is_admin) OR account_members.user_id = sqlc.arg(user_id);

-- name: ListScheduledTasksPage :many
SELECT scheduled_tasks.* FROM scheduled_tasks
JOIN account_members ON account_members.account_id = scheduled_tasks.account_id
WHERE sqlc.arg(is_admin) OR account_members.user_id = sqlc.arg(user_id)
GROUP BY scheduled_tasks.id
ORDER BY scheduled_tasks.created_at ASC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: GetScheduledTask :one
SELECT * FROM scheduled_tasks WHERE id = ? LIMIT 1;

-- name: ListAccountScheduledTasks :many
SELECT * FROM scheduled_tasks WHERE account_id = ? ORDER BY created_at ASC;

-- name: CreateScheduledTask :one
INSERT INTO scheduled_tasks (
    id, account_id, node_id, name, schedule, command, scheduler_driver, kind, enabled, status,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?)
RETURNING *;

-- name: UpdateScheduledTask :one
UPDATE scheduled_tasks
SET name = ?, schedule = ?, command = ?, scheduler_driver = ?, kind = ?, enabled = ?, status = 'pending', updated_at = ?
WHERE id = ?
RETURNING *;

-- name: UpdateScheduledTaskStatus :exec
UPDATE scheduled_tasks SET status = ?, updated_at = ? WHERE id = ?;

-- name: UpdateAccountTaskStatuses :exec
UPDATE scheduled_tasks
SET status = CASE WHEN enabled = 1 THEN ? ELSE ? END, updated_at = ?
WHERE account_id = ?;

-- name: DeleteScheduledTask :exec
DELETE FROM scheduled_tasks WHERE id = ?;
