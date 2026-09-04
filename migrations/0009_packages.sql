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

CREATE TABLE account_packages (
    account_id TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    package_id TEXT NOT NULL REFERENCES packages(id) ON DELETE RESTRICT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX account_packages_package_id_idx ON account_packages(package_id);

INSERT INTO packages (
    id,
    name,
    max_websites,
    max_domains,
    max_aliases,
    max_databases,
    max_database_users,
    max_scheduled_tasks,
    max_backup_plans,
    max_backup_retention,
    created_at,
    updated_at
) VALUES (
    '00000000000000000000000000000001',
    'Default',
    10,
    10,
    20,
    10,
    10,
    20,
    5,
    7,
    unixepoch(),
    unixepoch()
);

INSERT INTO account_packages (account_id, package_id, created_at, updated_at)
SELECT id, '00000000000000000000000000000001', unixepoch(), unixepoch()
FROM accounts;

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
    (SELECT COUNT(*) FROM cron_jobs WHERE cron_jobs.account_id = accounts.id) AS used_scheduled_tasks,
    (SELECT COUNT(*) FROM backup_plans WHERE backup_plans.account_id = accounts.id) AS used_backup_plans
FROM accounts
JOIN account_packages ON account_packages.account_id = accounts.id
JOIN packages ON packages.id = account_packages.package_id;
