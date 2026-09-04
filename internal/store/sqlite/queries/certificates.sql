-- name: ListCertificates :many
SELECT certificates.* FROM certificates
LEFT JOIN domains ON domains.id = certificates.domain_id
LEFT JOIN account_members ON account_members.account_id = domains.account_id
WHERE ? OR certificates.kind = 'panel' OR account_members.user_id = ?
GROUP BY certificates.id
ORDER BY certificates.created_at DESC;

-- name: CountCertificates :one
SELECT COUNT(DISTINCT certificates.id) FROM certificates
LEFT JOIN domains ON domains.id = certificates.domain_id
LEFT JOIN account_members ON account_members.account_id = domains.account_id
WHERE sqlc.arg(is_admin) OR certificates.kind = 'panel' OR account_members.user_id = sqlc.arg(user_id);

-- name: ListCertificatesPage :many
SELECT certificates.* FROM certificates
LEFT JOIN domains ON domains.id = certificates.domain_id
LEFT JOIN account_members ON account_members.account_id = domains.account_id
WHERE sqlc.arg(is_admin) OR certificates.kind = 'panel' OR account_members.user_id = sqlc.arg(user_id)
GROUP BY certificates.id
ORDER BY certificates.created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: GetCertificate :one
SELECT * FROM certificates WHERE id = ? LIMIT 1;

-- name: GetDomainCertificate :one
SELECT * FROM certificates WHERE domain_id = ? LIMIT 1;

-- name: GetPanelCertificate :one
SELECT * FROM certificates WHERE kind = 'panel' LIMIT 1;

-- name: CreateCertificate :one
INSERT INTO certificates (
    id, domain_id, node_id, kind, name, email, status, redirect_https,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, 'pending', ?, ?, ?)
RETURNING *;

-- name: ReplaceCertificateName :exec
INSERT OR IGNORE INTO certificate_names (certificate_id, name) VALUES (?, ?);

-- name: DeleteCertificateNames :exec
DELETE FROM certificate_names WHERE certificate_id = ?;

-- name: ListCertificateNames :many
SELECT name FROM certificate_names WHERE certificate_id = ? ORDER BY name;

-- name: QueueCertificate :exec
UPDATE certificates
SET name = ?, email = ?, redirect_https = ?, status = 'pending', error = '', updated_at = ?
WHERE id = ?;

-- name: SetCertificateResult :exec
UPDATE certificates
SET status = ?, expires_at = ?, renew_after = ?, error = ?, updated_at = ?
WHERE id = ?;

-- name: ListDueCertificates :many
SELECT * FROM certificates
WHERE status = 'active' AND renew_after IS NOT NULL AND renew_after <= ?
ORDER BY renew_after ASC;

-- name: CertificatePendingJobExists :one
SELECT EXISTS(
    SELECT 1 FROM jobs
    WHERE kind = 'certificate.renew' AND status IN ('queued', 'running')
      AND json_extract(payload, '$.certificateId') = ?
);

-- name: DeleteCertificate :exec
DELETE FROM certificates WHERE id = ?;
