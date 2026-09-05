-- name: CreatePackage :one
INSERT INTO packages (
    id, name, max_websites, max_domains, max_aliases, max_databases,
    max_database_users, max_scheduled_tasks, max_backup_plans,
    max_backup_retention, max_ftp_accounts, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: PackageNameExists :one
SELECT EXISTS(
    SELECT 1 FROM packages
    WHERE name = sqlc.arg(name) COLLATE NOCASE
      AND id != sqlc.arg(id)
);

-- name: GetPackageOverview :one
SELECT * FROM package_overviews WHERE id = ? LIMIT 1;

-- name: CountPackages :one
SELECT COUNT(*) FROM packages;

-- name: ListPackageOverviewsPage :many
SELECT * FROM package_overviews
ORDER BY name COLLATE NOCASE ASC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: UpdatePackage :one
UPDATE packages
SET
    name = ?,
    max_websites = ?,
    max_domains = ?,
    max_aliases = ?,
    max_databases = ?,
    max_database_users = ?,
    max_scheduled_tasks = ?,
    max_backup_plans = ?,
    max_backup_retention = ?,
    max_ftp_accounts = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeletePackage :execrows
DELETE FROM packages
WHERE id = ?
  AND NOT EXISTS (
      SELECT 1 FROM account_packages WHERE account_packages.package_id = packages.id
  );

-- name: AssignAccountPackage :exec
INSERT INTO account_packages (account_id, package_id, created_at, updated_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(account_id) DO UPDATE SET
    package_id = excluded.package_id,
    updated_at = excluded.updated_at;

-- name: GetAccountOverview :one
SELECT * FROM account_overviews WHERE id = ? LIMIT 1;

-- name: ListAccountOverviewsPage :many
SELECT * FROM account_overviews
ORDER BY created_at ASC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: ListUserAccountOverviewsPage :many
SELECT account_overviews.*
FROM account_overviews
JOIN account_members ON account_members.account_id = account_overviews.id
WHERE account_members.user_id = sqlc.arg(user_id)
ORDER BY account_overviews.created_at ASC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);
