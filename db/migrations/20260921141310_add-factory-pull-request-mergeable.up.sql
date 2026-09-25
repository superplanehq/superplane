BEGIN;

ALTER TABLE factory_pull_requests
  ADD COLUMN mergeable BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN merge_blocked_reason TEXT NOT NULL DEFAULT '',
  ADD COLUMN merge_blocked_message TEXT NOT NULL DEFAULT '',
  ADD COLUMN mergeable_head_sha TEXT NOT NULL DEFAULT '';

COMMIT;
