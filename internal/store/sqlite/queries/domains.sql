-- name: DomainNameExists :one
SELECT EXISTS(
    SELECT 1
    FROM domains
    WHERE domains.name = sqlc.arg(name) COLLATE NOCASE
       OR domains.previous_name = sqlc.arg(name) COLLATE NOCASE
    UNION ALL
    SELECT 1
    FROM domain_aliases
    WHERE domain_aliases.name = sqlc.arg(name) COLLATE NOCASE
       OR domain_aliases.previous_name = sqlc.arg(name) COLLATE NOCASE
);

-- name: CreateDomain :one
INSERT INTO domains (
    id,
    account_id,
    node_id,
    name,
    status,
    php_version,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, 'pending', ?, ?, ?)
RETURNING *;

-- name: GetDomain :one
SELECT * FROM domains WHERE id = ? LIMIT 1;

-- name: ListDomains :many
SELECT * FROM domains ORDER BY created_at ASC;

-- name: ListUserDomains :many
SELECT domains.*
FROM domains
JOIN account_members ON account_members.account_id = domains.account_id
WHERE account_members.user_id = ?
ORDER BY domains.created_at ASC;

-- name: UpdateDomainStatus :exec
UPDATE domains
SET status = ?, updated_at = ?
WHERE id = ?;

-- name: CreateDomainAlias :one
INSERT INTO domain_aliases (
    id,
    domain_id,
    name,
    status,
    created_at,
    updated_at
) VALUES (?, ?, ?, 'pending', ?, ?)
RETURNING *;

-- name: GetDomainAlias :one
SELECT * FROM domain_aliases WHERE id = ? LIMIT 1;

-- name: ListDomainAliases :many
SELECT *
FROM domain_aliases
WHERE domain_id = ?
ORDER BY created_at ASC;

-- name: ListEnabledDomainAliases :many
SELECT *
FROM domain_aliases
WHERE domain_id = ? AND enabled = 1
ORDER BY name ASC;

-- name: UpdateDomainAliasStatus :exec
UPDATE domain_aliases
SET status = ?, updated_at = ?
WHERE id = ?;

-- name: QueueDomainAction :one
UPDATE domains
SET enabled = ?, status = 'pending', updated_at = ?
WHERE id = ? AND status != 'pending'
RETURNING *;

-- name: QueueDomainAliasAction :one
UPDATE domain_aliases
SET enabled = ?, status = 'pending', updated_at = ?
WHERE id = ? AND status != 'pending'
RETURNING *;

-- name: DeleteDomain :exec
DELETE FROM domains WHERE id = ?;

-- name: DeleteDomainAlias :exec
DELETE FROM domain_aliases WHERE id = ?;

-- name: QueueDomainRename :one
UPDATE domains
SET previous_name = name, name = ?, status = 'pending', updated_at = ?
WHERE id = ? AND status = 'active'
RETURNING *;

-- name: QueueDomainAliasRename :one
UPDATE domain_aliases
SET previous_name = name, name = ?, status = 'pending', updated_at = ?
WHERE id = ? AND status != 'pending'
RETURNING *;

-- name: CompleteDomainRename :exec
UPDATE domains
SET previous_name = NULL, status = 'active', updated_at = ?
WHERE id = ?;

-- name: CompleteDomainAliasRename :exec
UPDATE domain_aliases
SET previous_name = NULL,
    status = CASE WHEN enabled = 1 THEN 'active' ELSE 'disabled' END,
    updated_at = ?
WHERE id = ?;

-- name: FailDomainRename :exec
UPDATE domains
SET name = COALESCE(previous_name, name), previous_name = NULL,
    status = 'error', updated_at = ?
WHERE id = ?;

-- name: FailDomainAliasRename :exec
UPDATE domain_aliases
SET name = COALESCE(previous_name, name), previous_name = NULL,
    status = 'error', updated_at = ?
WHERE id = ?;
