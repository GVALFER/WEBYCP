-- name: GetServiceSettings :one
SELECT * FROM service_settings WHERE id = 1;

-- name: UpdateServiceSettings :one
UPDATE service_settings
SET web_driver = ?,
    runtime_driver = ?,
    runtime_version = ?,
    database_driver = ?,
    scheduler_driver = ?,
    backup_driver = ?,
    updated_at = ?
WHERE id = 1
RETURNING *;
