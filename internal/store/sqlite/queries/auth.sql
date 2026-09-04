-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CreateUser :one
INSERT INTO users (
    id,
    username,
    email,
    name,
    password_hash,
    role,
    must_change_password,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = ? LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ? LIMIT 1;

-- name: UsernameExistsExcept :one
SELECT EXISTS(
    SELECT 1 FROM users WHERE username = ? AND id != ?
);

-- name: EmailExistsExcept :one
SELECT EXISTS(
    SELECT 1 FROM users WHERE email = ? AND id != ?
);

-- name: UpdateUserProfile :one
UPDATE users
SET username = ?, name = ?, email = ?, timezone = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: UpdateUserPassword :one
UPDATE users
SET password_hash = ?, must_change_password = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteOtherUserSessions :exec
DELETE FROM sessions WHERE user_id = ? AND id != ?;

-- name: DeleteUserSessions :exec
DELETE FROM sessions WHERE user_id = ?;

-- name: CreateSession :one
INSERT INTO sessions (
    id,
    user_id,
    token_hash,
    csrf_token,
    expires_at,
    created_at
) VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetSessionByTokenHash :one
SELECT * FROM sessions
WHERE token_hash = ? AND expires_at > ?
LIMIT 1;

-- name: DeleteSessionByTokenHash :exec
DELETE FROM sessions WHERE token_hash = ?;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions WHERE expires_at <= ?;
