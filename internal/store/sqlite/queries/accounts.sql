-- name: AccountNameExists :one
SELECT EXISTS(
    SELECT 1 FROM accounts WHERE name = ? COLLATE NOCASE
);

-- name: CreateAccount :one
INSERT INTO accounts (
    id,
    node_id,
    name,
    system_user,
    status,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, 'pending', ?, ?)
RETURNING *;

-- name: AddAccountMember :exec
INSERT INTO account_members (
    account_id,
    user_id,
    role,
    created_at
) VALUES (?, ?, 'owner', ?);

-- name: GetAccount :one
SELECT * FROM accounts WHERE id = ? LIMIT 1;

-- name: AccountMemberExists :one
SELECT EXISTS(
    SELECT 1 FROM account_members WHERE account_id = ? AND user_id = ?
);

-- name: ListAccounts :many
SELECT * FROM accounts ORDER BY created_at ASC;

-- name: ListUserAccounts :many
SELECT accounts.*
FROM accounts
JOIN account_members ON account_members.account_id = accounts.id
WHERE account_members.user_id = ?
ORDER BY accounts.created_at ASC;

-- name: UpdateAccountStatus :exec
UPDATE accounts
SET status = ?, updated_at = ?
WHERE id = ?;

-- name: QueueAccountAction :one
UPDATE accounts
SET enabled = ?, status = 'pending', updated_at = ?
WHERE id = ? AND status != 'pending'
RETURNING *;

-- name: AccountResourceCount :one
SELECT
    (SELECT COUNT(*) FROM domains WHERE domains.account_id = sqlc.arg(account_id)) +
    (SELECT COUNT(*) FROM databases WHERE databases.account_id = sqlc.arg(account_id)) +
    (SELECT COUNT(*) FROM database_users WHERE database_users.account_id = sqlc.arg(account_id)) +
    (SELECT COUNT(*) FROM cron_jobs WHERE cron_jobs.account_id = sqlc.arg(account_id)) +
    (SELECT COUNT(*) FROM backup_plans WHERE backup_plans.account_id = sqlc.arg(account_id)) +
    (SELECT COUNT(*) FROM backup_artifacts WHERE backup_artifacts.account_id = sqlc.arg(account_id));

-- name: DeleteAccount :exec
DELETE FROM accounts WHERE id = ?;
