-- name: ListDatabases :many
SELECT databases.* FROM databases
JOIN account_members ON account_members.account_id = databases.account_id
WHERE ? OR account_members.user_id = ?
GROUP BY databases.id
ORDER BY databases.created_at ASC;

-- name: CountDatabases :one
SELECT COUNT(DISTINCT databases.id) FROM databases
JOIN account_members ON account_members.account_id = databases.account_id
WHERE sqlc.arg(is_admin) OR account_members.user_id = sqlc.arg(user_id);

-- name: ListDatabasesPage :many
SELECT databases.* FROM databases
JOIN account_members ON account_members.account_id = databases.account_id
WHERE sqlc.arg(is_admin) OR account_members.user_id = sqlc.arg(user_id)
GROUP BY databases.id
ORDER BY databases.created_at ASC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: GetDatabase :one
SELECT * FROM databases WHERE id = ? LIMIT 1;

-- name: DatabaseNameExists :one
SELECT EXISTS(SELECT 1 FROM databases WHERE account_id = ? AND name = ? COLLATE NOCASE);

-- name: CreateDatabase :one
INSERT INTO databases (
    id, account_id, node_id, name, system_name, driver, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, 'pending', ?, ?)
RETURNING *;

-- name: UpdateDatabaseStatus :exec
UPDATE databases SET status = ?, updated_at = ? WHERE id = ?;

-- name: QueueDatabaseDelete :one
UPDATE databases SET status = 'pending', updated_at = ? WHERE id = ? RETURNING *;

-- name: DeleteDatabase :exec
DELETE FROM databases WHERE id = ?;

-- name: ListDatabaseUsers :many
SELECT database_users.* FROM database_users
JOIN account_members ON account_members.account_id = database_users.account_id
WHERE ? OR account_members.user_id = ?
GROUP BY database_users.id
ORDER BY database_users.created_at ASC;

-- name: CountDatabaseUsers :one
SELECT COUNT(DISTINCT database_users.id) FROM database_users
JOIN account_members ON account_members.account_id = database_users.account_id
WHERE sqlc.arg(is_admin) OR account_members.user_id = sqlc.arg(user_id);

-- name: ListDatabaseUsersPage :many
SELECT database_users.* FROM database_users
JOIN account_members ON account_members.account_id = database_users.account_id
WHERE sqlc.arg(is_admin) OR account_members.user_id = sqlc.arg(user_id)
GROUP BY database_users.id
ORDER BY database_users.created_at ASC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: GetDatabaseUser :one
SELECT * FROM database_users WHERE id = ? LIMIT 1;

-- name: DatabaseUserNameExists :one
SELECT EXISTS(SELECT 1 FROM database_users WHERE account_id = ? AND name = ? COLLATE NOCASE);

-- name: CreateDatabaseUser :one
INSERT INTO database_users (
    id, account_id, node_id, name, system_name, driver, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, 'pending', ?, ?)
RETURNING *;

-- name: UpdateDatabaseUserStatus :exec
UPDATE database_users SET status = ?, updated_at = ? WHERE id = ?;

-- name: QueueDatabaseUserDelete :one
UPDATE database_users SET status = 'pending', updated_at = ? WHERE id = ? RETURNING *;

-- name: DeleteDatabaseUser :exec
DELETE FROM database_users WHERE id = ?;

-- name: ListDatabaseGrants :many
SELECT database_grants.* FROM database_grants
JOIN databases ON databases.id = database_grants.database_id
JOIN account_members ON account_members.account_id = databases.account_id
WHERE ? OR account_members.user_id = ?
GROUP BY database_grants.database_id, database_grants.database_user_id
ORDER BY database_grants.created_at ASC;

-- name: CountDatabaseGrants :one
SELECT COUNT(*) FROM (
    SELECT database_grants.database_id, database_grants.database_user_id
    FROM database_grants
    JOIN databases ON databases.id = database_grants.database_id
    JOIN account_members ON account_members.account_id = databases.account_id
    WHERE sqlc.arg(is_admin) OR account_members.user_id = sqlc.arg(user_id)
    GROUP BY database_grants.database_id, database_grants.database_user_id
);

-- name: ListDatabaseGrantsPage :many
SELECT database_grants.* FROM database_grants
JOIN databases ON databases.id = database_grants.database_id
JOIN account_members ON account_members.account_id = databases.account_id
WHERE sqlc.arg(is_admin) OR account_members.user_id = sqlc.arg(user_id)
GROUP BY database_grants.database_id, database_grants.database_user_id
ORDER BY database_grants.created_at ASC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: GetDatabaseGrant :one
SELECT * FROM database_grants WHERE database_id = ? AND database_user_id = ? LIMIT 1;

-- name: UpsertDatabaseGrant :one
INSERT INTO database_grants (
    database_id, database_user_id, status, created_at, updated_at
) VALUES (?, ?, 'pending', ?, ?)
ON CONFLICT(database_id, database_user_id) DO UPDATE SET
    status = 'pending', updated_at = excluded.updated_at
RETURNING *;

-- name: UpdateDatabaseGrantStatus :exec
UPDATE database_grants SET status = ?, updated_at = ?
WHERE database_id = ? AND database_user_id = ?;

-- name: QueueDatabaseGrantDelete :one
UPDATE database_grants SET status = 'pending', updated_at = ?
WHERE database_id = ? AND database_user_id = ?
RETURNING *;

-- name: DeleteDatabaseGrant :exec
DELETE FROM database_grants WHERE database_id = ? AND database_user_id = ?;

-- name: ListAccountDatabaseSystemNames :many
SELECT system_name FROM databases
WHERE account_id = ? AND status = 'active'
ORDER BY system_name;
