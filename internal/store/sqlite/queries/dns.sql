-- name: EnsureDNSProvider :one
INSERT INTO dns_providers (
    id, node_id, name, driver, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT (node_id, driver) DO UPDATE SET name = excluded.name
RETURNING *;

-- name: ListDNSProviders :many
SELECT * FROM dns_providers ORDER BY created_at ASC;

-- name: GetDNSProvider :one
SELECT * FROM dns_providers WHERE id = ? LIMIT 1;

-- name: GetDNSSettings :one
SELECT * FROM dns_settings WHERE id = 1;

-- name: UpdateDNSSettings :one
UPDATE dns_settings
SET primary_nameserver = ?, secondary_nameserver = ?, default_ttl = ?, updated_at = ?
WHERE id = 1
RETURNING *;

-- name: CreateDNSZone :one
INSERT INTO dns_zones (
    id, account_id, node_id, provider_id, name, primary_nameserver,
    secondary_nameserver, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?)
RETURNING *;

-- name: GetDNSZone :one
SELECT * FROM dns_zones WHERE id = ? LIMIT 1;

-- name: CountDNSZones :one
SELECT COUNT(*) FROM dns_zones;

-- name: CountUserDNSZones :one
SELECT COUNT(*) FROM dns_zones
WHERE EXISTS (
    SELECT 1 FROM account_members
    WHERE account_members.account_id = dns_zones.account_id
      AND account_members.user_id = ?
);

-- name: ListDNSZonesPage :many
SELECT * FROM dns_zones
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: ListUserDNSZonesPage :many
SELECT * FROM dns_zones
WHERE EXISTS (
    SELECT 1 FROM account_members
    WHERE account_members.account_id = dns_zones.account_id
      AND account_members.user_id = sqlc.arg(user_id)
)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: QueueDNSZoneDelete :one
UPDATE dns_zones
SET status = 'deleting', updated_at = ?
WHERE dns_zones.id = ?
  AND dns_zones.status NOT IN ('pending', 'deleting')
  AND NOT EXISTS (
      SELECT 1 FROM dns_records
      WHERE dns_records.zone_id = dns_zones.id
        AND dns_records.status IN ('pending', 'deleting')
  )
RETURNING *;

-- name: UpdateDNSZoneStatus :exec
UPDATE dns_zones SET status = ?, updated_at = ? WHERE id = ?;

-- name: DeleteDNSZone :exec
DELETE FROM dns_zones WHERE id = ?;

-- name: CreateDNSRecord :one
INSERT INTO dns_records (
    id, zone_id, name, type, content, ttl, priority, synced_name, synced_type,
    status, created_at, updated_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, '', '', 'pending', ?, ?
WHERE EXISTS (
    SELECT 1 FROM dns_zones
    WHERE dns_zones.id = sqlc.arg(zone_id) AND dns_zones.status = 'active'
)
RETURNING *;

-- name: GetDNSRecord :one
SELECT * FROM dns_records WHERE id = ? LIMIT 1;

-- name: CountDNSRecords :one
SELECT COUNT(*) FROM dns_records WHERE zone_id = ?;

-- name: ListDNSRecordsPage :many
SELECT * FROM dns_records
WHERE zone_id = sqlc.arg(zone_id)
ORDER BY name ASC, type ASC, content ASC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: ListDNSRecords :many
SELECT * FROM dns_records
WHERE zone_id = ?
ORDER BY name ASC, type ASC, content ASC;

-- name: ListDNSRecordsByName :many
SELECT * FROM dns_records
WHERE zone_id = ? AND name = ? COLLATE NOCASE;

-- name: QueueDNSRecordUpdate :one
UPDATE dns_records
SET name = ?, type = ?, content = ?, ttl = ?, priority = ?, status = 'pending', updated_at = ?
WHERE dns_records.id = ?
  AND dns_records.status NOT IN ('pending', 'deleting')
  AND EXISTS (
      SELECT 1 FROM dns_zones
      WHERE dns_zones.id = dns_records.zone_id AND dns_zones.status = 'active'
  )
RETURNING *;

-- name: QueueDNSRecordDelete :one
UPDATE dns_records
SET status = 'deleting', updated_at = ?
WHERE dns_records.id = ?
  AND dns_records.status NOT IN ('pending', 'deleting')
  AND EXISTS (
      SELECT 1 FROM dns_zones
      WHERE dns_zones.id = dns_records.zone_id AND dns_zones.status = 'active'
  )
RETURNING *;

-- name: UpdateDNSRecordStatus :exec
UPDATE dns_records SET status = ?, updated_at = ? WHERE id = ?;

-- name: CompleteDNSRecordSync :exec
UPDATE dns_records
SET status = 'active', synced_name = name, synced_type = type, updated_at = ?
WHERE id = ?;

-- name: DeleteDNSRecord :exec
DELETE FROM dns_records WHERE id = ?;
