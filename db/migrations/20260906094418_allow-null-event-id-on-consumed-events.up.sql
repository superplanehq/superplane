ALTER TABLE workflow_node_execution_consumed_events
    ALTER COLUMN event_id DROP NOT NULL;

ALTER TABLE workflow_node_execution_consumed_events
    DROP CONSTRAINT workflow_node_execution_consumed_events_event_id_fkey;

ALTER TABLE workflow_node_execution_consumed_events
    ADD CONSTRAINT workflow_node_execution_consumed_events_event_id_fkey
    FOREIGN KEY (event_id)
    REFERENCES workflow_events(id)
    ON DELETE SET NULL;
