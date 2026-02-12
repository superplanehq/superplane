begin;

ALTER TABLE workflow_node_requests
  ADD COLUMN attempts INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN retry_strategy JSONB NOT NULL DEFAULT '{}',
  ADD COLUMN result CHARACTER VARYING(32),
  ADD COLUMN result_message TEXT;

commit;