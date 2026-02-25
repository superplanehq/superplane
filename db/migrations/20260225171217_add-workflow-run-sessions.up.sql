ALTER TABLE workflow_events
ADD COLUMN continuation_key TEXT;

CREATE INDEX idx_workflow_events_workflow_continuation_key
ON workflow_events (workflow_id, continuation_key)
WHERE continuation_key IS NOT NULL;

CREATE TABLE workflow_run_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    continuation_key TEXT NOT NULL,
    root_event_id UUID NOT NULL REFERENCES workflow_events(id) ON DELETE CASCADE,
    last_execution_id UUID REFERENCES workflow_node_executions(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workflow_id, continuation_key),
    UNIQUE (workflow_id, root_event_id)
);

CREATE INDEX idx_workflow_run_sessions_workflow_root
ON workflow_run_sessions (workflow_id, root_event_id);
