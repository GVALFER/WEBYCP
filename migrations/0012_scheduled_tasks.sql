ALTER TABLE cron_jobs RENAME TO scheduled_tasks;
ALTER TABLE scheduled_tasks ADD COLUMN kind TEXT NOT NULL DEFAULT 'command' CHECK (kind IN ('command'));
DROP INDEX cron_jobs_account_id_idx;
CREATE INDEX scheduled_tasks_account_id_idx ON scheduled_tasks(account_id);

UPDATE jobs SET kind = 'task.sync' WHERE kind = 'cron.sync';
UPDATE job_steps SET name = 'task.sync' WHERE name = 'cron.sync';

ALTER TABLE audit_events ADD COLUMN job_id TEXT;
DROP INDEX audit_events_created_at_idx;
CREATE INDEX audit_events_created_at_idx ON audit_events(created_at DESC, id DESC);
CREATE INDEX audit_events_job_id_idx ON audit_events(job_id);
