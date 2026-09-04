ALTER TABLE nodes ADD COLUMN capabilities TEXT;
ALTER TABLE nodes ADD COLUMN capabilities_at INTEGER;

ALTER TABLE databases ADD COLUMN driver TEXT NOT NULL DEFAULT 'mysql';
ALTER TABLE database_users ADD COLUMN driver TEXT NOT NULL DEFAULT 'mysql';
ALTER TABLE cron_jobs ADD COLUMN scheduler_driver TEXT NOT NULL DEFAULT 'crontab';
ALTER TABLE backup_plans ADD COLUMN storage_driver TEXT NOT NULL DEFAULT 'local';
ALTER TABLE backup_runs ADD COLUMN storage_driver TEXT NOT NULL DEFAULT 'local';
ALTER TABLE backup_artifacts ADD COLUMN storage_driver TEXT NOT NULL DEFAULT 'local';

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
    id,
    web_driver,
    runtime_driver,
    runtime_version,
    database_driver,
    scheduler_driver,
    backup_driver,
    updated_at
) VALUES (1, 'nginx', 'phpfpm', '8.3', 'mysql', 'crontab', 'local', unixepoch() * 1000);

UPDATE packages
SET created_at = created_at * 1000,
    updated_at = updated_at * 1000
WHERE created_at < 100000000000;

UPDATE account_packages
SET created_at = created_at * 1000,
    updated_at = updated_at * 1000
WHERE created_at < 100000000000;
