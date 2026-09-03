-- name: ListDatabases :many
SELECT databases.* FROM databases
JOIN account_members ON account_members.account_id = databases.account_id
WHERE ? OR account_members.user_id = ?
GROUP BY databases.id
ORDER BY databases.created_at ASC;

-- name: GetDatabase :one
SELECT * FROM databases WHERE id = ? LIMIT 1;

-- name: DatabaseNameExists :one
SELECT EXISTS(SELECT 1 FROM databases WHERE account_id = ? AND name = ? COLLATE NOCASE);

-- name: CreateDatabase :one
INSERT INTO databases (
    id, account_id, node_id, name, system_name, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, 'pending', ?, ?)
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

-- name: GetDatabaseUser :one
SELECT * FROM database_users WHERE id = ? LIMIT 1;

-- name: DatabaseUserNameExists :one
SELECT EXISTS(SELECT 1 FROM database_users WHERE account_id = ? AND name = ? COLLATE NOCASE);

-- name: CreateDatabaseUser :one
INSERT INTO database_users (
    id, account_id, node_id, name, system_name, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, 'pending', ?, ?)
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
