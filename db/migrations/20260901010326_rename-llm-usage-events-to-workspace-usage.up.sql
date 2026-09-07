BEGIN;

ALTER TABLE llm_usage_events RENAME TO workspace_usage_events;

ALTER INDEX llm_usage_events_pkey RENAME TO workspace_usage_events_pkey;
ALTER INDEX llm_usage_events_idempotency_key_key RENAME TO workspace_usage_events_idempotency_key_key;
ALTER INDEX idx_llm_usage_events_org_occurred RENAME TO idx_workspace_usage_events_org_occurred;
ALTER INDEX idx_llm_usage_events_factory_occurred RENAME TO idx_workspace_usage_events_factory_occurred;
ALTER INDEX idx_llm_usage_events_work_order RENAME TO idx_workspace_usage_events_work_order;
ALTER INDEX idx_llm_usage_events_execution RENAME TO idx_workspace_usage_events_execution;

ALTER TABLE workspace_usage_events
  ADD COLUMN duration_seconds BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN machine_type TEXT NOT NULL DEFAULT '',
  ADD COLUMN fleet_id TEXT NOT NULL DEFAULT '';

ALTER TABLE workspace_usage_events
  ADD CONSTRAINT workspace_usage_events_usage_kind_known CHECK (usage_kind IN ('model', 'compute'));

ALTER TABLE workspace_usage_events
  ADD CONSTRAINT workspace_usage_events_compute_shape
  CHECK (usage_kind <> 'compute' OR (machine_type <> '' AND duration_seconds >= 0));

CREATE INDEX idx_workspace_usage_events_compute_org_occurred
  ON workspace_usage_events (organization_id, occurred_at DESC)
  WHERE usage_kind = 'compute';

CREATE INDEX idx_workspace_usage_events_compute_factory_occurred
  ON workspace_usage_events (factory_id, occurred_at DESC)
  WHERE usage_kind = 'compute' AND factory_id IS NOT NULL;

CREATE INDEX idx_workspace_usage_events_compute_machine
  ON workspace_usage_events (organization_id, machine_type, occurred_at DESC)
  WHERE usage_kind = 'compute';

ALTER TABLE factory_work_order_executions
  ADD COLUMN duration_seconds BIGINT NOT NULL DEFAULT 0;

COMMIT;
