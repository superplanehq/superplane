ALTER TABLE factory_work_order_checks
  ADD COLUMN repository character varying(255) NOT NULL DEFAULT '';

DROP INDEX IF EXISTS idx_factory_work_order_checks_order_key_unique;

CREATE UNIQUE INDEX idx_factory_work_order_checks_order_repository_key_unique
  ON factory_work_order_checks USING btree (work_order_id, repository, key);
