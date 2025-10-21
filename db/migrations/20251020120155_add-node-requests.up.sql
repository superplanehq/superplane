begin;

CREATE TABLE workflow_node_requests (
  id              uuid NOT NULL DEFAULT uuid_generate_v4(),
  workflow_id     uuid NOT NULL,
  node_id         CHARACTER VARYING(128) NOT NULL,
  execution_id    uuid,
  state           CHARACTER VARYING(32) NOT NULL,
  type            CHARACTER VARYING(32) NOT NULL,
  spec            JSONB NOT NULL,
  run_at          TIMESTAMP NOT NULL,
  created_at      TIMESTAMP NOT NULL,
  updated_at      TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE,
  FOREIGN KEY (workflow_id, node_id) REFERENCES workflow_nodes(workflow_id, node_id) ON DELETE CASCADE,
  FOREIGN KEY (execution_id) REFERENCES workflow_node_executions(id) ON DELETE CASCADE
);

CREATE INDEX idx_node_requests_state_run_at ON workflow_node_requests(state, run_at) WHERE state = 'pending';

commit;
