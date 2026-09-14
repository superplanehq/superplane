ALTER TABLE installation_metadata
  ADD COLUMN IF NOT EXISTS max_parallel_factory_tasks INTEGER NOT NULL DEFAULT 50;

ALTER TABLE installation_metadata
  ADD CONSTRAINT installation_metadata_max_parallel_factory_tasks_positive
  CHECK (max_parallel_factory_tasks >= 1);

ALTER TABLE organizations
  ADD COLUMN IF NOT EXISTS max_parallel_factory_tasks INTEGER;

ALTER TABLE organizations
  ADD CONSTRAINT organizations_max_parallel_factory_tasks_positive
  CHECK (max_parallel_factory_tasks IS NULL OR max_parallel_factory_tasks >= 1);

CREATE INDEX IF NOT EXISTS idx_factory_work_order_executions_factory_active
  ON public.factory_work_order_executions USING btree (factory_id)
  WHERE ((status)::text = ANY ((ARRAY['pending'::character varying, 'running'::character varying])::text[]));
