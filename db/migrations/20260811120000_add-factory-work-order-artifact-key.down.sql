BEGIN;

DROP INDEX IF EXISTS idx_factory_work_order_artifacts_factory_key_unique;

ALTER TABLE factory_work_order_artifacts
  DROP COLUMN IF EXISTS key;

COMMIT;
