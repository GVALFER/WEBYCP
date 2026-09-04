-- name: WebsiteHostnameExists :one
SELECT EXISTS(
    SELECT 1 FROM website_domains
    WHERE hostname = sqlc.arg(hostname) COLLATE NOCASE
       OR previous_hostname = sqlc.arg(hostname) COLLATE NOCASE
);

-- name: CreateWebsite :one
INSERT INTO websites (
    id, account_id, node_id, name, kind, document_root, web_driver,
    runtime_driver, runtime_version, status, enabled, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', 1, ?, ?)
RETURNING *;

-- name: CreateWebsiteDomain :one
INSERT INTO website_domains (
    id, website_id, hostname, kind, status, enabled, created_at, updated_at
) VALUES (?, ?, ?, ?, 'pending', 1, ?, ?)
RETURNING *;

-- name: GetWebsite :one
SELECT * FROM websites WHERE id = ? LIMIT 1;

-- name: GetWebsiteDomain :one
SELECT * FROM website_domains WHERE id = ? LIMIT 1;

-- name: GetWebsitePrimaryDomain :one
SELECT * FROM website_domains WHERE website_id = ? AND kind = 'primary' LIMIT 1;

-- name: ListWebsites :many
SELECT * FROM websites ORDER BY created_at ASC;

-- name: CountWebsites :one
SELECT COUNT(*) FROM websites;

-- name: ListWebsitesPage :many
SELECT * FROM websites ORDER BY created_at ASC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: ListUserWebsites :many
SELECT websites.* FROM websites
JOIN account_members ON account_members.account_id = websites.account_id
WHERE account_members.user_id = ?
ORDER BY websites.created_at ASC;

-- name: CountUserWebsites :one
SELECT COUNT(*) FROM websites
JOIN account_members ON account_members.account_id = websites.account_id
WHERE account_members.user_id = ?;

-- name: ListUserWebsitesPage :many
SELECT websites.* FROM websites
JOIN account_members ON account_members.account_id = websites.account_id
WHERE account_members.user_id = sqlc.arg(user_id)
ORDER BY websites.created_at ASC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: ListWebsiteDomains :many
SELECT * FROM website_domains WHERE website_id = ? ORDER BY kind DESC, hostname ASC;

-- name: ListEnabledWebsiteDomains :many
SELECT * FROM website_domains
WHERE website_id = ? AND enabled = 1
ORDER BY kind DESC, hostname ASC;

-- name: CountWebsiteDomainsByKind :one
SELECT COUNT(*) FROM website_domains
JOIN websites ON websites.id = website_domains.website_id
WHERE website_domains.kind = sqlc.arg(kind)
  AND (sqlc.arg(is_admin) OR EXISTS (
      SELECT 1 FROM account_members
      WHERE account_members.account_id = websites.account_id
        AND account_members.user_id = sqlc.arg(user_id)
  ));

-- name: ListWebsiteDomainsByKindPage :many
SELECT website_domains.* FROM website_domains
JOIN websites ON websites.id = website_domains.website_id
WHERE website_domains.kind = sqlc.arg(kind)
  AND (sqlc.arg(is_admin) OR EXISTS (
      SELECT 1 FROM account_members
      WHERE account_members.account_id = websites.account_id
        AND account_members.user_id = sqlc.arg(user_id)
  ))
ORDER BY website_domains.created_at ASC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: QueueWebsiteAction :one
UPDATE websites SET enabled = ?, status = 'pending', updated_at = ?
WHERE id = ? AND status != 'pending'
RETURNING *;

-- name: QueueWebsiteDomainAction :one
UPDATE website_domains SET enabled = ?, status = 'pending', updated_at = ?
WHERE id = ? AND kind = 'alias' AND status != 'pending'
RETURNING *;

-- name: QueueWebsiteDomainRename :one
UPDATE website_domains
SET previous_hostname = hostname, hostname = ?, status = 'pending', updated_at = ?
WHERE id = ? AND status != 'pending'
RETURNING *;

-- name: UpdateWebsiteStatus :exec
UPDATE websites SET status = ?, updated_at = ? WHERE id = ?;

-- name: UpdateWebsiteDomainStatus :exec
UPDATE website_domains SET status = ?, updated_at = ? WHERE id = ?;

-- name: CompleteWebsiteDomainRename :exec
UPDATE website_domains
SET previous_hostname = NULL,
    status = CASE WHEN enabled = 1 THEN 'active' ELSE 'disabled' END,
    updated_at = ?
WHERE id = ?;

-- name: FailWebsiteDomainRename :exec
UPDATE website_domains
SET hostname = COALESCE(previous_hostname, hostname), previous_hostname = NULL,
    status = 'error', updated_at = ?
WHERE id = ?;

-- name: DeleteWebsite :exec
DELETE FROM websites WHERE id = ?;

-- name: DeleteWebsiteDomain :exec
DELETE FROM website_domains WHERE id = ? AND kind = 'alias';
