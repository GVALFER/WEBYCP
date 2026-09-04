-- name: CreateNode :one
INSERT INTO nodes (
    id,
    name,
    kind,
    endpoint,
    status,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetNode :one
SELECT * FROM nodes WHERE id = ? LIMIT 1;

-- name: GetLocalNode :one
SELECT * FROM nodes WHERE kind = 'local' LIMIT 1;

-- name: ListNodes :many
SELECT * FROM nodes ORDER BY created_at ASC;

-- name: UpdateLocalNode :one
UPDATE nodes
SET name = ?, endpoint = ?, updated_at = ?
WHERE kind = 'local'
RETURNING *;

-- name: UpdateNodeProbe :exec
UPDATE nodes
SET status = sqlc.arg(status),
    last_seen_at = COALESCE(sqlc.narg(last_seen_at), last_seen_at),
    capabilities = COALESCE(sqlc.narg(capabilities), capabilities),
    capabilities_at = COALESCE(sqlc.narg(capabilities_at), capabilities_at),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
