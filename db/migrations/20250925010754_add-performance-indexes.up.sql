BEGIN;

CREATE INDEX IF NOT EXISTS idx_stage_executions_stage_id_state_created_at ON stage_executions(stage_id, state, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_stage_executions_id_stage_id ON stage_executions(id, stage_id);

CREATE INDEX IF NOT EXISTS idx_stage_events_stage_id_state_created_at ON stage_events(stage_id, state, created_at ASC);

COMMIT;
