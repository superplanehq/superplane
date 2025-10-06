begin;

CREATE TABLE workflows (
  id              uuid NOT NULL DEFAULT uuid_generate_v4(),
  organization_id uuid NOT NULL,
  name            CHARACTER VARYING(128) NOT NULL,
  description     TEXT,
  created_at      TIMESTAMP NOT NULL,
  updated_at      TIMESTAMP NOT NULL,
  nodes           JSONB NOT NULL DEFAULT '[]'::jsonb,
  edges           JSONB NOT NULL DEFAULT '[]'::jsonb,

  PRIMARY KEY (id),
  UNIQUE (organization_id, name)
);

CREATE INDEX idx_workflows_organization_id ON workflows(organization_id);

CREATE TABLE workflow_queues (
  workflow_id uuid NOT NULL,
  node_id     CHARACTER VARYING(128) NOT NULL,
  data        JSONB NOT NULL,

  PRIMARY KEY (workflow_id, node_id),
  FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE
);

CREATE INDEX idx_workflow_queues_workflow_node_id ON workflow_queues(workflow_id, node_id);

CREATE TABLE workflow_node_executions (
  workflow_id    uuid NOT NULL,
  node_id        CHARACTER VARYING(128) NOT NULL,
  state          CHARACTER VARYING(32) NOT NULL,
  result         CHARACTER VARYING(32),
  result_reason  CHARACTER VARYING(128),
  result_message TEXT,
  input          JSONB,
  output         JSONB,

  PRIMARY KEY (workflow_id, node_id),
  FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE
);

CREATE INDEX idx_workflow_node_executions_workflow_id ON workflow_node_executions(workflow_id, node_id);

commit;
