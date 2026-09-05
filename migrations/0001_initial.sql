-- Initial public schema. Timestamps are UTC Unix milliseconds.

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL COLLATE NOCASE UNIQUE,
    name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'user')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    username TEXT NOT NULL COLLATE NOCASE DEFAULT '',
    must_change_password INTEGER NOT NULL DEFAULT 0 CHECK (must_change_password IN (0, 1)),
    timezone TEXT NOT NULL DEFAULT 'UTC'
);

CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    csrf_token TEXT NOT NULL,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE TABLE accounts (
    id TEXT PRIMARY KEY,
    node_id TEXT REFERENCES nodes(id) ON DELETE SET NULL,
    name TEXT NOT NULL UNIQUE,
    system_user TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'disabled', 'error')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1))
);

CREATE TABLE account_members (
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('owner', 'member')),
    created_at INTEGER NOT NULL,
    PRIMARY KEY (account_id, user_id)
);

CREATE TABLE nodes (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('local', 'remote')),
    endpoint TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('unknown', 'online', 'offline')),
    last_seen_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    capabilities TEXT,
    capabilities_at INTEGER
);

CREATE TABLE jobs (
    id TEXT PRIMARY KEY,
    node_id TEXT REFERENCES nodes(id) ON DELETE SET NULL,
    user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    kind TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed')),
    payload TEXT NOT NULL DEFAULT '{}',
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 1,
    error TEXT,
    created_at INTEGER NOT NULL,
    started_at INTEGER,
    finished_at INTEGER
);

CREATE TABLE job_steps (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('running', 'succeeded', 'failed')),
    message TEXT NOT NULL DEFAULT '',
    started_at INTEGER NOT NULL,
    finished_at INTEGER
);

CREATE TABLE audit_events (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    result TEXT NOT NULL CHECK (result IN ('success', 'failure')),
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at INTEGER NOT NULL,
    job_id TEXT
);

CREATE TABLE databases (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    name TEXT NOT NULL COLLATE NOCASE,
    system_name TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'error')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    driver TEXT NOT NULL DEFAULT 'mysql',
    UNIQUE (account_id, name)
);

CREATE TABLE database_users (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    name TEXT NOT NULL COLLATE NOCASE,
    system_name TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'error')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    driver TEXT NOT NULL DEFAULT 'mysql',
    UNIQUE (account_id, name)
);

CREATE TABLE database_grants (
    database_id TEXT NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
    database_user_id TEXT NOT NULL REFERENCES database_users(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'error')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (database_id, database_user_id)
);

CREATE TABLE scheduled_tasks (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    schedule TEXT NOT NULL,
    command TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'disabled', 'error')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    scheduler_driver TEXT NOT NULL DEFAULT 'crontab',
    kind TEXT NOT NULL DEFAULT 'command' CHECK (kind IN ('command'))
);

CREATE TABLE backup_plans (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    schedule TEXT NOT NULL DEFAULT '',
    retention_count INTEGER NOT NULL DEFAULT 7 CHECK (retention_count BETWEEN 1 AND 100),
    include_files INTEGER NOT NULL DEFAULT 1 CHECK (include_files IN (0, 1)),
    include_databases INTEGER NOT NULL DEFAULT 1 CHECK (include_databases IN (0, 1)),
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    last_run_at INTEGER,
    next_run_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    storage_driver TEXT NOT NULL DEFAULT 'local'
);

CREATE TABLE backup_runs (
    id TEXT PRIMARY KEY,
    plan_id TEXT REFERENCES backup_plans(id) ON DELETE SET NULL,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed')),
    error TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    started_at INTEGER,
    finished_at INTEGER,
    storage_driver TEXT NOT NULL DEFAULT 'local'
);

CREATE TABLE backup_artifacts (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL UNIQUE REFERENCES backup_runs(id) ON DELETE CASCADE,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    path TEXT NOT NULL UNIQUE,
    checksum TEXT NOT NULL,
    size_bytes INTEGER NOT NULL CHECK (size_bytes >= 0),
    manifest TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    storage_driver TEXT NOT NULL DEFAULT 'local'
);

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

CREATE TABLE certificates (
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

CREATE TABLE certificate_names (
    certificate_id TEXT NOT NULL REFERENCES certificates(id) ON DELETE CASCADE,
    name TEXT NOT NULL COLLATE NOCASE,
    PRIMARY KEY (certificate_id, name)
);

CREATE TABLE packages (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL COLLATE NOCASE UNIQUE,
    max_websites INTEGER NOT NULL CHECK (max_websites BETWEEN 0 AND 1000000),
    max_domains INTEGER NOT NULL CHECK (max_domains BETWEEN 0 AND 1000000),
    max_aliases INTEGER NOT NULL CHECK (max_aliases BETWEEN 0 AND 1000000),
    max_databases INTEGER NOT NULL CHECK (max_databases BETWEEN 0 AND 1000000),
    max_database_users INTEGER NOT NULL CHECK (max_database_users BETWEEN 0 AND 1000000),
    max_scheduled_tasks INTEGER NOT NULL CHECK (max_scheduled_tasks BETWEEN 0 AND 1000000),
    max_backup_plans INTEGER NOT NULL CHECK (max_backup_plans BETWEEN 0 AND 1000000),
    max_backup_retention INTEGER NOT NULL CHECK (max_backup_retention BETWEEN 1 AND 100),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

INSERT INTO packages (
    id, name, max_websites, max_domains, max_aliases, max_databases,
    max_database_users, max_scheduled_tasks, max_backup_plans,
    max_backup_retention, created_at, updated_at
) VALUES (
    '00000000000000000000000000000001', 'Default',
    10, 10, 20, 10, 10, 20, 5, 7, unixepoch() * 1000, unixepoch() * 1000
);

CREATE TABLE account_packages (
    account_id TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    package_id TEXT NOT NULL REFERENCES packages(id) ON DELETE RESTRICT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE service_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    web_driver TEXT NOT NULL,
    runtime_driver TEXT NOT NULL,
    runtime_version TEXT NOT NULL,
    database_driver TEXT NOT NULL,
    scheduler_driver TEXT NOT NULL,
    backup_driver TEXT NOT NULL,
    updated_at INTEGER NOT NULL
);

INSERT INTO service_settings (
    id, web_driver, runtime_driver, runtime_version, database_driver,
    scheduler_driver, backup_driver, updated_at
) VALUES (1, 'nginx', 'phpfpm', '8.3', 'mysql', 'crontab', 'local', unixepoch() * 1000);

CREATE TABLE dns_providers (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    driver TEXT NOT NULL CHECK (driver IN ('powerdns')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    UNIQUE (node_id, driver)
);

CREATE TABLE dns_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    primary_nameserver TEXT NOT NULL,
    secondary_nameserver TEXT NOT NULL,
    default_ttl INTEGER NOT NULL CHECK (default_ttl BETWEEN 60 AND 86400),
    updated_at INTEGER NOT NULL
);

INSERT INTO dns_settings (
    id, primary_nameserver, secondary_nameserver, default_ttl, updated_at
) VALUES (1, '', '', 3600, unixepoch() * 1000);

CREATE TABLE dns_zones (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    provider_id TEXT NOT NULL REFERENCES dns_providers(id) ON DELETE RESTRICT,
    name TEXT NOT NULL COLLATE NOCASE UNIQUE,
    primary_nameserver TEXT NOT NULL,
    secondary_nameserver TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'deleting', 'error')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE dns_records (
    id TEXT PRIMARY KEY,
    zone_id TEXT NOT NULL REFERENCES dns_zones(id) ON DELETE CASCADE,
    name TEXT NOT NULL COLLATE NOCASE,
    type TEXT NOT NULL CHECK (type IN ('A', 'AAAA', 'CNAME', 'MX', 'TXT')),
    content TEXT NOT NULL,
    ttl INTEGER NOT NULL CHECK (ttl BETWEEN 60 AND 86400),
    priority INTEGER NOT NULL CHECK (priority BETWEEN 0 AND 65535),
    synced_name TEXT NOT NULL,
    synced_type TEXT NOT NULL CHECK (synced_type IN ('', 'A', 'AAAA', 'CNAME', 'MX', 'TXT')),
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'deleting', 'error')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    UNIQUE (zone_id, name, type, content, priority)
);

CREATE VIEW package_overviews AS
SELECT
    packages.*,
    COUNT(account_packages.account_id) AS account_count
FROM packages
LEFT JOIN account_packages ON account_packages.package_id = packages.id
GROUP BY packages.id;

CREATE VIEW account_overviews AS
SELECT
    accounts.id,
    accounts.node_id,
    accounts.name,
    accounts.system_user,
    accounts.status,
    accounts.created_at,
    accounts.updated_at,
    accounts.enabled,
    packages.id AS package_id,
    packages.name AS package_name,
    packages.created_at AS package_created_at,
    packages.updated_at AS package_updated_at,
    (
        SELECT COUNT(*)
        FROM account_packages AS package_accounts
        WHERE package_accounts.package_id = packages.id
    ) AS package_account_count,
    packages.max_websites,
    packages.max_domains,
    packages.max_aliases,
    packages.max_databases,
    packages.max_database_users,
    packages.max_scheduled_tasks,
    packages.max_backup_plans,
    packages.max_backup_retention,
    (SELECT COUNT(*) FROM websites WHERE websites.account_id = accounts.id) AS used_websites,
    (
        SELECT COUNT(*)
        FROM website_domains
        JOIN websites ON websites.id = website_domains.website_id
        WHERE websites.account_id = accounts.id AND website_domains.kind = 'primary'
    ) AS used_domains,
    (
        SELECT COUNT(*)
        FROM website_domains
        JOIN websites ON websites.id = website_domains.website_id
        WHERE websites.account_id = accounts.id AND website_domains.kind = 'alias'
    ) AS used_aliases,
    (SELECT COUNT(*) FROM databases WHERE databases.account_id = accounts.id) AS used_databases,
    (SELECT COUNT(*) FROM database_users WHERE database_users.account_id = accounts.id) AS used_database_users,
    (SELECT COUNT(*) FROM scheduled_tasks WHERE scheduled_tasks.account_id = accounts.id) AS used_scheduled_tasks,
    (SELECT COUNT(*) FROM backup_plans WHERE backup_plans.account_id = accounts.id) AS used_backup_plans
FROM accounts
JOIN account_packages ON account_packages.account_id = accounts.id
JOIN packages ON packages.id = account_packages.package_id;

CREATE INDEX sessions_user_id_idx ON sessions(user_id);

CREATE INDEX sessions_expires_at_idx ON sessions(expires_at);

CREATE UNIQUE INDEX nodes_local_kind_idx ON nodes(kind) WHERE kind = 'local';

CREATE INDEX jobs_status_created_at_idx ON jobs(status, created_at);

CREATE INDEX jobs_node_id_idx ON jobs(node_id);

CREATE INDEX job_steps_job_id_idx ON job_steps(job_id, started_at);

CREATE INDEX audit_events_user_id_idx ON audit_events(user_id);

CREATE INDEX databases_account_id_idx ON databases(account_id);

CREATE INDEX database_users_account_id_idx ON database_users(account_id);

CREATE INDEX backup_plans_account_id_idx ON backup_plans(account_id);

CREATE INDEX backup_plans_next_run_idx ON backup_plans(enabled, next_run_at);

CREATE INDEX backup_runs_plan_id_idx ON backup_runs(plan_id, created_at);

CREATE UNIQUE INDEX backup_runs_active_plan_idx
ON backup_runs(plan_id) WHERE plan_id IS NOT NULL AND status IN ('queued', 'running');

CREATE INDEX backup_artifacts_account_id_idx ON backup_artifacts(account_id, created_at);

CREATE UNIQUE INDEX users_username_idx ON users(username);

CREATE INDEX websites_account_id_idx ON websites(account_id);

CREATE UNIQUE INDEX website_domains_primary_idx
ON website_domains(website_id) WHERE kind = 'primary';

CREATE INDEX website_domains_website_id_idx ON website_domains(website_id, kind);

CREATE UNIQUE INDEX certificates_website_id_idx
ON certificates(website_id) WHERE website_id IS NOT NULL;

CREATE UNIQUE INDEX certificates_panel_kind_idx
ON certificates(kind) WHERE kind = 'panel';

CREATE INDEX certificates_renew_after_idx ON certificates(status, renew_after);

CREATE INDEX account_packages_package_id_idx ON account_packages(package_id);

CREATE INDEX dns_zones_account_id_idx ON dns_zones(account_id);

CREATE INDEX dns_zones_provider_id_idx ON dns_zones(provider_id);

CREATE INDEX dns_records_zone_id_idx ON dns_records(zone_id);

CREATE INDEX dns_records_rrset_idx ON dns_records(zone_id, name, type);

CREATE INDEX scheduled_tasks_account_id_idx ON scheduled_tasks(account_id);

CREATE INDEX audit_events_created_at_idx ON audit_events(created_at DESC, id DESC);

CREATE INDEX audit_events_job_id_idx ON audit_events(job_id);
