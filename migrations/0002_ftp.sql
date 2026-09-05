ALTER TABLE packages ADD COLUMN max_ftp_accounts INTEGER NOT NULL DEFAULT 10
    CHECK (max_ftp_accounts BETWEEN 0 AND 100);

DROP VIEW package_overviews;
CREATE VIEW package_overviews AS
SELECT packages.*, COUNT(account_packages.account_id) AS account_count
FROM packages
LEFT JOIN account_packages ON account_packages.package_id = packages.id
GROUP BY packages.id;

CREATE TABLE ftp_accounts (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    username TEXT NOT NULL COLLATE NOCASE,
    password_hash TEXT NOT NULL,
    enabled INTEGER NOT NULL CHECK (enabled IN (0, 1)),
    deleting INTEGER NOT NULL DEFAULT 0 CHECK (deleting IN (0, 1)),
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'disabled', 'error')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    UNIQUE (node_id, username)
);

CREATE INDEX ftp_accounts_account_id_idx ON ftp_accounts(account_id);

CREATE VIEW ftp_overviews AS
SELECT ftp_accounts.*, accounts.name AS account_name,
    accounts.system_user, accounts.status AS account_status
FROM ftp_accounts JOIN accounts ON accounts.id = ftp_accounts.account_id;

DROP VIEW account_overviews;
CREATE VIEW account_overviews AS
SELECT
    accounts.id, accounts.node_id, accounts.name, accounts.system_user,
    accounts.status, accounts.created_at, accounts.updated_at, accounts.enabled,
    packages.id AS package_id, packages.name AS package_name,
    packages.created_at AS package_created_at, packages.updated_at AS package_updated_at,
    (SELECT COUNT(*) FROM account_packages AS package_accounts
        WHERE package_accounts.package_id = packages.id) AS package_account_count,
    packages.max_websites, packages.max_domains, packages.max_aliases,
    packages.max_databases, packages.max_database_users, packages.max_scheduled_tasks,
    packages.max_backup_plans, packages.max_backup_retention, packages.max_ftp_accounts,
    (SELECT COUNT(*) FROM websites WHERE websites.account_id = accounts.id) AS used_websites,
    (SELECT COUNT(*) FROM website_domains JOIN websites ON websites.id = website_domains.website_id
        WHERE websites.account_id = accounts.id AND website_domains.kind = 'primary') AS used_domains,
    (SELECT COUNT(*) FROM website_domains JOIN websites ON websites.id = website_domains.website_id
        WHERE websites.account_id = accounts.id AND website_domains.kind = 'alias') AS used_aliases,
    (SELECT COUNT(*) FROM databases WHERE databases.account_id = accounts.id) AS used_databases,
    (SELECT COUNT(*) FROM database_users WHERE database_users.account_id = accounts.id) AS used_database_users,
    (SELECT COUNT(*) FROM scheduled_tasks WHERE scheduled_tasks.account_id = accounts.id) AS used_scheduled_tasks,
    (SELECT COUNT(*) FROM backup_plans WHERE backup_plans.account_id = accounts.id) AS used_backup_plans,
    (SELECT COUNT(*) FROM ftp_accounts WHERE ftp_accounts.account_id = accounts.id) AS used_ftp_accounts
FROM accounts
JOIN account_packages ON account_packages.account_id = accounts.id
JOIN packages ON packages.id = account_packages.package_id;
