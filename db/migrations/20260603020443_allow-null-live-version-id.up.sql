BEGIN;

-- Canvas provisioning creates the workflow row before the first live version is materialized from git.
ALTER TABLE workflows
  ALTER COLUMN live_version_id DROP NOT NULL;

COMMIT;
