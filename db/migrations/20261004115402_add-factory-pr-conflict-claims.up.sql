BEGIN;

--
-- One row records the pull request head that already started a conflict
-- repair for a handler. A new head can start another repair. The same head
-- does not start a second run.
--
CREATE TABLE factory_pr_conflict_claims (
  handler_id      UUID NOT NULL REFERENCES factory_pr_feedback_handlers(id) ON DELETE CASCADE,
  pull_request_id UUID NOT NULL REFERENCES factory_pull_requests(id) ON DELETE CASCADE,
  head_sha        TEXT NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (handler_id, pull_request_id),
  CONSTRAINT factory_pr_conflict_claims_head_sha_present CHECK (btrim(head_sha) <> '')
);

COMMIT;
