BEGIN;

CREATE TABLE factory_pull_request_revisions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  pull_request_id UUID NOT NULL
    REFERENCES factory_pull_requests(id) ON DELETE RESTRICT,
  sha TEXT NOT NULL,
  observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT factory_pull_request_revisions_sha_present CHECK (btrim(sha) <> '')
);

CREATE UNIQUE INDEX idx_factory_pull_request_revisions_pull_request_sha
  ON factory_pull_request_revisions (pull_request_id, sha);

CREATE UNIQUE INDEX idx_factory_pull_request_revisions_id_pull_request
  ON factory_pull_request_revisions (id, pull_request_id);

CREATE INDEX idx_factory_pull_request_revisions_observed
  ON factory_pull_request_revisions (pull_request_id, observed_at DESC);

ALTER TABLE factory_pull_requests
  ADD COLUMN current_revision_id UUID,
  ADD COLUMN active_mutation_run_id UUID
    REFERENCES workflow_runs(id) ON DELETE SET NULL;

ALTER TABLE factory_pull_requests
  ADD CONSTRAINT factory_pull_requests_current_revision_fk
  FOREIGN KEY (current_revision_id)
  REFERENCES factory_pull_request_revisions(id)
  ON DELETE RESTRICT;

ALTER TABLE factory_pull_request_runs
  ADD COLUMN feedback_handler_id UUID
    REFERENCES factory_pr_feedback_handlers(id) ON DELETE SET NULL,
  ADD COLUMN revision_id UUID,
  ADD COLUMN access VARCHAR(32) NOT NULL DEFAULT 'concurrent',
  ADD COLUMN state VARCHAR(32) NOT NULL DEFAULT 'active',
  ADD COLUMN attempt INTEGER,
  ADD COLUMN attempt_limit INTEGER,
  ADD COLUMN access_requested_at TIMESTAMPTZ,
  ADD COLUMN access_granted_at TIMESTAMPTZ,
  ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE factory_pull_request_runs AS links
SET
  state = CASE
    WHEN runs.state = 'finished' THEN 'finished'
    ELSE 'active'
  END,
  access = CASE
    WHEN runs.state = 'finished' THEN 'released'
    ELSE 'concurrent'
  END,
  updated_at = links.created_at
FROM workflow_runs AS runs
WHERE runs.id = links.run_id;

ALTER TABLE factory_pull_request_runs
  ADD CONSTRAINT factory_pull_request_runs_access_valid
    CHECK (access IN ('concurrent', 'waiting', 'exclusive', 'released')),
  ADD CONSTRAINT factory_pull_request_runs_state_valid
    CHECK (state IN ('active', 'finished', 'limit_reached')),
  ADD CONSTRAINT factory_pull_request_runs_attempt_positive
    CHECK (attempt IS NULL OR attempt > 0),
  ADD CONSTRAINT factory_pull_request_runs_attempt_limit_positive
    CHECK (attempt_limit IS NULL OR attempt_limit > 0),
  ADD CONSTRAINT factory_pull_request_runs_revision_fk
    FOREIGN KEY (revision_id, pull_request_id)
    REFERENCES factory_pull_request_revisions(id, pull_request_id);

CREATE UNIQUE INDEX idx_factory_pull_request_runs_active_handler_revision
  ON factory_pull_request_runs (pull_request_id, revision_id, feedback_handler_id)
  WHERE revision_id IS NOT NULL
    AND feedback_handler_id IS NOT NULL
    AND state = 'active';

CREATE INDEX idx_factory_pull_request_runs_exclusive_queue
  ON factory_pull_request_runs (pull_request_id, access, access_requested_at);

CREATE INDEX idx_factory_pull_request_runs_attempt_history
  ON factory_pull_request_runs (feedback_handler_id, pull_request_id, access_granted_at DESC);

ALTER TABLE factory_pr_feedback_handlers
  ADD COLUMN maximum_attempts INTEGER,
  ADD CONSTRAINT factory_pr_feedback_handlers_maximum_attempts_positive
    CHECK (maximum_attempts IS NULL OR maximum_attempts > 0);

COMMIT;
