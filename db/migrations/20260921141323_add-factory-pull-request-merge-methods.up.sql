BEGIN;

ALTER TABLE factory_pull_requests
  ADD COLUMN mergeable_allowed_methods TEXT NOT NULL DEFAULT '';

COMMIT;
