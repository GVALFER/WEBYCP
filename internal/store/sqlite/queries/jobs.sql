-- name: CreateJob :one
INSERT INTO jobs (
    id,
    node_id,
    user_id,
    kind,
    status,
    payload,
    attempts,
    max_attempts,
    created_at
) VALUES (?, ?, ?, ?, 'queued', ?, 0, ?, ?)
RETURNING *;

-- name: GetJob :one
SELECT * FROM jobs WHERE id = ? LIMIT 1;

-- name: ListJobs :many
SELECT * FROM jobs
ORDER BY created_at DESC
LIMIT ?;

-- name: CountJobs :one
SELECT COUNT(*) FROM jobs;

-- name: ListJobsPage :many
SELECT * FROM jobs
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: ClaimJob :one
UPDATE jobs
SET status = 'running', attempts = attempts + 1, started_at = ?, error = NULL
WHERE id = (
    SELECT id FROM jobs
    WHERE status = 'queued' AND attempts < max_attempts
    ORDER BY created_at ASC
    LIMIT 1
)
RETURNING *;

-- name: CompleteJob :exec
UPDATE jobs
SET status = 'succeeded', finished_at = ?, error = NULL,
    payload = json_remove(payload, '$.password')
WHERE id = ? AND status = 'running';

-- name: RetryJob :exec
UPDATE jobs
SET status = 'queued', started_at = NULL, finished_at = NULL, error = ?
WHERE id = ? AND status = 'running' AND attempts < max_attempts;

-- name: FailJob :exec
UPDATE jobs
SET status = 'failed', finished_at = ?, error = ?,
    payload = json_remove(payload, '$.password')
WHERE id = ? AND status = 'running';

-- name: RecoverJobs :execrows
UPDATE jobs
SET status = 'queued',
    attempts = CASE WHEN attempts > 0 THEN attempts - 1 ELSE 0 END,
    started_at = NULL,
    error = NULL
WHERE status = 'running';

-- name: CreateJobStep :one
INSERT INTO job_steps (
    id,
    job_id,
    name,
    status,
    message,
    started_at
) VALUES (?, ?, ?, 'running', '', ?)
RETURNING *;

-- name: FinishJobStep :exec
UPDATE job_steps
SET status = ?, message = ?, finished_at = ?
WHERE id = ?;

-- name: ListJobSteps :many
SELECT * FROM job_steps WHERE job_id = ? ORDER BY started_at ASC;
