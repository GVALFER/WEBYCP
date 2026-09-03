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
SET status = ?, last_seen_at = ?, updated_at = ?
WHERE id = ?;
