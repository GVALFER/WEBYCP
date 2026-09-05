-- name: GetFTP :one
SELECT * FROM ftp_overviews WHERE id = ?;

-- name: AccountFTP :many
SELECT * FROM ftp_overviews WHERE account_id = ? ORDER BY created_at, id;

-- name: CountFTP :one
SELECT COUNT(*) FROM ftp_accounts
WHERE sqlc.arg(is_admin) OR EXISTS (
    SELECT 1 FROM account_members
    WHERE account_members.account_id = ftp_accounts.account_id AND user_id = sqlc.arg(user_id)
);

-- name: ListFTPPage :many
SELECT * FROM ftp_overviews
WHERE sqlc.arg(is_admin) OR EXISTS (
    SELECT 1 FROM account_members
    WHERE account_members.account_id = ftp_overviews.account_id AND user_id = sqlc.arg(user_id)
)
ORDER BY created_at, id
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: FTPNameExists :one
SELECT EXISTS (
    SELECT 1 FROM ftp_accounts
    WHERE node_id = ? AND username = ? COLLATE NOCASE AND id != ?
);

-- name: FTPBusy :one
SELECT EXISTS (
    SELECT 1 FROM jobs WHERE kind = 'ftp.sync' AND status IN ('queued', 'running')
    AND json_extract(payload, '$.accountId') = sqlc.arg(account_id)
);

-- name: CreateFTP :exec
INSERT INTO ftp_accounts (id, account_id, node_id, username, password_hash, enabled, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, 'pending', ?, ?);

-- name: UpdateFTP :exec
UPDATE ftp_accounts SET username = ?, password_hash = ?, enabled = ?, deleting = ?,
    status = 'pending', updated_at = ? WHERE id = ?;

-- name: DeleteFTP :exec
DELETE FROM ftp_accounts WHERE account_id = ? AND deleting = 1;

-- name: SetFTPStatuses :exec
UPDATE ftp_accounts SET status = CASE
    WHEN CAST(sqlc.arg(failed) AS INTEGER) = 1 THEN 'error'
    WHEN enabled = 1 THEN 'active' ELSE 'disabled' END, updated_at = sqlc.arg(updated_at)
WHERE account_id = sqlc.arg(account_id);
