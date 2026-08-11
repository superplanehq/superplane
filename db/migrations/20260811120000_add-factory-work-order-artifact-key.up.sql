BEGIN;

--
-- Optional, per-factory-unique key on work order artifacts so callers can
-- look up a work order from an external identifier (e.g. a PR url) without
-- having to already know the order id. NULL keys are excluded from the
-- uniqueness check so most artifacts (which don't set one) never collide.
--
ALTER TABLE factory_work_order_artifacts
  ADD COLUMN key VARCHAR(512);

CREATE UNIQUE INDEX idx_factory_work_order_artifacts_factory_key_unique
  ON factory_work_order_artifacts (factory_id, key)
  WHERE key IS NOT NULL;

COMMIT;
