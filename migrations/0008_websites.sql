CREATE TABLE websites (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('php')),
    document_root TEXT NOT NULL,
    web_driver TEXT NOT NULL CHECK (web_driver IN ('nginx')),
    runtime_driver TEXT NOT NULL CHECK (runtime_driver IN ('phpfpm')),
    runtime_version TEXT NOT NULL CHECK (runtime_version IN ('8.3')),
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'disabled', 'error')),
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX websites_account_id_idx ON websites(account_id);

CREATE TABLE website_domains (
    id TEXT PRIMARY KEY,
    website_id TEXT NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    hostname TEXT NOT NULL COLLATE NOCASE UNIQUE,
    kind TEXT NOT NULL CHECK (kind IN ('primary', 'alias')),
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'disabled', 'error')),
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    previous_hostname TEXT COLLATE NOCASE,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX website_domains_primary_idx
ON website_domains(website_id) WHERE kind = 'primary';
CREATE INDEX website_domains_website_id_idx ON website_domains(website_id, kind);

INSERT INTO websites (
    id, account_id, node_id, name, kind, document_root, web_driver,
    runtime_driver, runtime_version, status, enabled, created_at, updated_at
)
SELECT
    domains.id,
    domains.account_id,
    domains.node_id,
    domains.name,
    'php',
    '/home/' || accounts.system_user || '/web/' || domains.name || '/public_html',
    'nginx',
    'phpfpm',
    domains.php_version,
    domains.status,
    domains.enabled,
    domains.created_at,
    domains.updated_at
FROM domains
JOIN accounts ON accounts.id = domains.account_id;

INSERT INTO website_domains (
    id, website_id, hostname, kind, status, enabled, previous_hostname,
    created_at, updated_at
)
SELECT
    id, id, name, 'primary', status, enabled, previous_name, created_at, updated_at
FROM domains;

INSERT INTO website_domains (
    id, website_id, hostname, kind, status, enabled, previous_hostname,
    created_at, updated_at
)
SELECT
    id, domain_id, name, 'alias', status, enabled, previous_name, created_at, updated_at
FROM domain_aliases;

CREATE TABLE certificates_new (
    id TEXT PRIMARY KEY,
    website_id TEXT REFERENCES websites(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    kind TEXT NOT NULL CHECK (kind IN ('website', 'panel')),
    name TEXT NOT NULL COLLATE NOCASE,
    email TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'error')),
    redirect_https INTEGER NOT NULL DEFAULT 1 CHECK (redirect_https IN (0, 1)),
    expires_at INTEGER,
    renew_after INTEGER,
    error TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    CHECK ((kind = 'website' AND website_id IS NOT NULL) OR (kind = 'panel' AND website_id IS NULL))
);

INSERT INTO certificates_new (
    id, website_id, node_id, kind, name, email, status, redirect_https,
    expires_at, renew_after, error, created_at, updated_at
)
SELECT
    id, domain_id, node_id,
    CASE WHEN kind = 'domain' THEN 'website' ELSE kind END,
    name, email, status, redirect_https, expires_at, renew_after, error,
    created_at, updated_at
FROM certificates;

CREATE TABLE certificate_names_new (
    certificate_id TEXT NOT NULL REFERENCES certificates_new(id) ON DELETE CASCADE,
    name TEXT NOT NULL COLLATE NOCASE,
    PRIMARY KEY (certificate_id, name)
);

INSERT INTO certificate_names_new (certificate_id, name)
SELECT certificate_id, name FROM certificate_names;

DROP TABLE certificate_names;
DROP TABLE certificates;
ALTER TABLE certificates_new RENAME TO certificates;
ALTER TABLE certificate_names_new RENAME TO certificate_names;

CREATE UNIQUE INDEX certificates_website_id_idx
ON certificates(website_id) WHERE website_id IS NOT NULL;
CREATE UNIQUE INDEX certificates_panel_kind_idx
ON certificates(kind) WHERE kind = 'panel';
CREATE INDEX certificates_renew_after_idx ON certificates(status, renew_after);

UPDATE jobs
SET kind = CASE kind
    WHEN 'domain.create' THEN 'website.create'
    WHEN 'domain.delete' THEN 'website.delete'
    WHEN 'domain.disable' THEN 'website.disable'
    WHEN 'domain.enable' THEN 'website.enable'
    WHEN 'domain.update' THEN 'website_domain.update'
    WHEN 'alias.create' THEN 'website_domain.create'
    WHEN 'alias.delete' THEN 'website_domain.delete'
    WHEN 'alias.disable' THEN 'website_domain.disable'
    WHEN 'alias.enable' THEN 'website_domain.enable'
    WHEN 'alias.update' THEN 'website_domain.update'
    ELSE kind
END,
payload = CASE
    WHEN kind IN ('domain.create', 'domain.delete', 'domain.disable', 'domain.enable')
        THEN replace(payload, '"domainId":', '"websiteId":')
    WHEN kind = 'domain.update'
        THEN replace(replace(replace(payload, '"domainId":', '"websiteDomainId":'), '"previousName":', '"previousHostname":'), '"name":', '"hostname":')
    WHEN kind IN ('alias.create', 'alias.delete', 'alias.disable', 'alias.enable')
        THEN replace(payload, '"aliasId":', '"websiteDomainId":')
    WHEN kind = 'alias.update'
        THEN replace(replace(replace(payload, '"aliasId":', '"websiteDomainId":'), '"previousName":', '"previousHostname":'), '"name":', '"hostname":')
    ELSE payload
END
WHERE kind IN (
    'domain.create', 'domain.delete', 'domain.disable', 'domain.enable', 'domain.update',
    'alias.create', 'alias.delete', 'alias.disable', 'alias.enable', 'alias.update'
);

UPDATE audit_events
SET action = CASE action
    WHEN 'domain.create' THEN 'website.create'
    WHEN 'domain.delete' THEN 'website.delete'
    WHEN 'domain.disable' THEN 'website.disable'
    WHEN 'domain.enable' THEN 'website.enable'
    WHEN 'domain.update' THEN 'website_domain.update'
    WHEN 'alias.create' THEN 'website_domain.create'
    WHEN 'alias.delete' THEN 'website_domain.delete'
    WHEN 'alias.disable' THEN 'website_domain.disable'
    WHEN 'alias.enable' THEN 'website_domain.enable'
    WHEN 'alias.update' THEN 'website_domain.update'
    ELSE action
END,
resource_type = CASE resource_type
    WHEN 'domain' THEN 'website'
    WHEN 'domain_alias' THEN 'website_domain'
    ELSE resource_type
END
WHERE resource_type IN ('domain', 'domain_alias');

DROP TABLE domain_aliases;
DROP TABLE domains;
