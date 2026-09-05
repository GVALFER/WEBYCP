-- name: CreateAuditEvent :exec
INSERT INTO audit_events (
    id,
    user_id,
    action,
    resource_type,
    resource_id,
    result,
    metadata,
    created_at,
    job_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: CountAuditEvents :one
SELECT COUNT(*) FROM audit_events
WHERE sqlc.arg(job_id) = '' OR job_id = sqlc.arg(job_id);

-- name: ListAuditEventsPage :many
SELECT id, user_id, action, resource_type, resource_id, result, created_at, job_id FROM audit_events
WHERE sqlc.arg(job_id) = '' OR job_id = sqlc.arg(job_id)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);
