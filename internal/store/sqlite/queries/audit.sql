-- name: CreateAuditEvent :exec
INSERT INTO audit_events (
    id,
    user_id,
    action,
    resource_type,
    resource_id,
    result,
    metadata,
    created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

