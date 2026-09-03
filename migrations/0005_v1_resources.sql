ALTER TABLE accounts
ADD COLUMN enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1));

CREATE TABLE certificates (
    id TEXT PRIMARY KEY,
    domain_id TEXT REFERENCES domains(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    kind TEXT NOT NULL CHECK (kind IN ('domain', 'panel')),
    name TEXT NOT NULL COLLATE NOCASE,
    email TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'error')),
    redirect_https INTEGER NOT NULL DEFAULT 1 CHECK (redirect_https IN (0, 1)),
    expires_at INTEGER,
    renew_after INTEGER,
    error TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    CHECK ((kind = 'domain' AND domain_id IS NOT NULL) OR (kind = 'panel' AND domain_id IS NULL))
);

CREATE UNIQUE INDEX certificates_domain_id_idx
ON certificates(domain_id) WHERE domain_id IS NOT NULL;
CREATE UNIQUE INDEX certificates_panel_kind_idx
ON certificates(kind) WHERE kind = 'panel';
CREATE INDEX certificates_renew_after_idx ON certificates(status, renew_after);

CREATE TABLE certificate_names (
    certificate_id TEXT NOT NULL REFERENCES certificates(id) ON DELETE CASCADE,
    name TEXT NOT NULL COLLATE NOCASE,
    PRIMARY KEY (certificate_id, name)
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
    UNIQUE (account_id, name)
);

CREATE INDEX databases_account_id_idx ON databases(account_id);

CREATE TABLE database_users (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    name TEXT NOT NULL COLLATE NOCASE,
    system_name TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'error')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    UNIQUE (account_id, name)
);

CREATE INDEX database_users_account_id_idx ON database_users(account_id);

CREATE TABLE database_grants (
    database_id TEXT NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
    database_user_id TEXT NOT NULL REFERENCES database_users(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'error')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (database_id, database_user_id)
);

CREATE TABLE cron_jobs (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    schedule TEXT NOT NULL,
    command TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'disabled', 'error')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX cron_jobs_account_id_idx ON cron_jobs(account_id);

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
    updated_at INTEGER NOT NULL
);

CREATE INDEX backup_plans_account_id_idx ON backup_plans(account_id);
CREATE INDEX backup_plans_next_run_idx ON backup_plans(enabled, next_run_at);

CREATE TABLE backup_runs (
    id TEXT PRIMARY KEY,
    plan_id TEXT REFERENCES backup_plans(id) ON DELETE SET NULL,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed')),
    error TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    started_at INTEGER,
    finished_at INTEGER
);

CREATE INDEX backup_runs_plan_id_idx ON backup_runs(plan_id, created_at);
CREATE UNIQUE INDEX backup_runs_active_plan_idx
ON backup_runs(plan_id) WHERE plan_id IS NOT NULL AND status IN ('queued', 'running');

CREATE TABLE backup_artifacts (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL UNIQUE REFERENCES backup_runs(id) ON DELETE CASCADE,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    path TEXT NOT NULL UNIQUE,
    checksum TEXT NOT NULL,
    size_bytes INTEGER NOT NULL CHECK (size_bytes >= 0),
    manifest TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE INDEX backup_artifacts_account_id_idx ON backup_artifacts(account_id, created_at);
