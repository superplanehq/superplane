ALTER TABLE factory_planning_sessions DROP CONSTRAINT factory_planning_sessions_kind_check;
ALTER TABLE factory_planning_sessions ADD CONSTRAINT factory_planning_sessions_kind_check CHECK (kind IN ('task_creation', 'work_order_analysis', 'custom_automation'));
ALTER TABLE factory_planning_sessions ADD COLUMN origin_run_id uuid;
CREATE UNIQUE INDEX idx_factory_planning_sessions_origin_run ON factory_planning_sessions (origin_run_id) WHERE origin_run_id IS NOT NULL;
