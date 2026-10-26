ALTER TABLE runner_tasks ADD COLUMN dispatch_started_at timestamptz;
CREATE TABLE runner_task_preparations (
 task_id uuid PRIMARY KEY REFERENCES runner_tasks(id),
 started_at timestamptz NOT NULL,
 deadline timestamptz NOT NULL,
 attempt_id uuid,
 attempt_expires_at timestamptz,
 attempt_count integer NOT NULL DEFAULT 0,
 next_retry_at timestamptz NOT NULL,
 prepared_at timestamptz,
 updated_at timestamptz NOT NULL
);
CREATE INDEX runner_task_preparation_expiry ON runner_task_preparations(deadline) WHERE prepared_at IS NULL;
CREATE TABLE runner_credential_cleanups (
 id uuid PRIMARY KEY,
 task_id uuid NOT NULL REFERENCES runner_tasks(id),
 attempt_id uuid NOT NULL,
 provider text NOT NULL,
 provider_resource_id uuid NOT NULL,
 name text NOT NULL UNIQUE,
 key_hash text NOT NULL DEFAULT '',
 expires_at timestamptz NOT NULL,
 accepted_at timestamptz,
 completed_at timestamptz,
 next_retry_at timestamptz NOT NULL,
 attempt_count integer NOT NULL DEFAULT 0,
 created_at timestamptz NOT NULL,
 updated_at timestamptz NOT NULL
);
CREATE INDEX runner_credential_cleanup_pending ON runner_credential_cleanups(next_retry_at) WHERE completed_at IS NULL;
