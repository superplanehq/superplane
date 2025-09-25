-- Add performance indexes for slow queries

-- Index for stage_executions queries filtering by stage_id and ordering by created_at
-- Used in stage.go:686 (DISTINCT ON queries with ORDER BY)
CREATE INDEX IF NOT EXISTS idx_stage_executions_stage_id_created_at ON stage_executions(stage_id, created_at DESC);

-- Composite index for stage_executions queries by id AND stage_id
-- Used in stage_execution.go:269 (FindExecutionByID)
CREATE INDEX IF NOT EXISTS idx_stage_executions_id_stage_id ON stage_executions(id, stage_id);

-- Composite index for stage_events queries by stage_id AND state with created_at ordering
-- Used in stage.go:734 (getQueueInfoForStages)
CREATE INDEX IF NOT EXISTS idx_stage_events_stage_id_state_created_at ON stage_events(stage_id, state, created_at ASC);

-- Composite index for connections queries by target_id AND target_type
-- Used in connection.go:81 (ListConnectionsInTransaction)
CREATE INDEX IF NOT EXISTS idx_connections_target_id_target_type ON connections(target_id, target_type);

-- Additional index for event_rejections queries by target_id and target_type
-- Used in event_rejection.go queries
CREATE INDEX IF NOT EXISTS idx_event_rejections_target_id_target_type ON event_rejections(target_id, target_type);