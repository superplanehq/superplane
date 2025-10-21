begin;

CREATE TABLE workflows (
  id              uuid NOT NULL DEFAULT uuid_generate_v4(),
  organization_id uuid NOT NULL,
  name            CHARACTER VARYING(128) NOT NULL,
  description     TEXT,
  created_at      TIMESTAMP NOT NULL,
  updated_at      TIMESTAMP NOT NULL,
  edges           JSONB NOT NULL DEFAULT '[]'::jsonb,

  PRIMARY KEY (id),
  UNIQUE (organization_id, name)
);

CREATE TABLE workflow_events (
  id            uuid NOT NULL DEFAULT uuid_generate_v4(),
  workflow_id   uuid NOT NULL,
  node_id       CHARACTER VARYING(128),
  channel       CHARACTER VARYING(64),
  data          JSONB NOT NULL,
  state         CHARACTER VARYING(32) NOT NULL,
  execution_id  uuid,
  created_at    TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE
);

CREATE TABLE workflow_nodes (
  workflow_id   uuid NOT NULL,
  node_id       CHARACTER VARYING(128) NOT NULL,
  name          CHARACTER VARYING(128) NOT NULL,
  state         CHARACTER VARYING(32) NOT NULL,
  type          CHARACTER VARYING(32) NOT NULL,
  ref           JSONB NOT NULL,
  configuration JSONB NOT NULL DEFAULT '{}'::jsonb,
  metadata      JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at    TIMESTAMP NOT NULL,
  updated_at    TIMESTAMP NOT NULL,

  PRIMARY KEY (workflow_id, node_id),
  FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE
);

CREATE TABLE workflow_node_queue_items (
  id            uuid NOT NULL DEFAULT uuid_generate_v4(),
  workflow_id   uuid NOT NULL,
  node_id       CHARACTER VARYING(128) NOT NULL,
  root_event_id uuid NOT NULL,
  event_id      uuid NOT NULL,
  created_at    TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE,
  FOREIGN KEY (root_event_id) REFERENCES workflow_events(id) ON DELETE CASCADE,
  FOREIGN KEY (event_id) REFERENCES workflow_events(id) ON DELETE CASCADE
);

CREATE TABLE workflow_node_executions (
  id                      uuid NOT NULL DEFAULT uuid_generate_v4(),
  workflow_id             uuid NOT NULL,
  node_id                 CHARACTER VARYING(128) NOT NULL,
  root_event_id           uuid NOT NULL,
  event_id                uuid NOT NULL,
  previous_execution_id   uuid,
  parent_execution_id     uuid,
  state                   CHARACTER VARYING(32) NOT NULL,
  result                  CHARACTER VARYING(32),
  result_reason           CHARACTER VARYING(128),
  result_message          TEXT,
  metadata                JSONB NOT NULL DEFAULT '{}'::jsonb,
  configuration           JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at              TIMESTAMP NOT NULL,
  updated_at              TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE,
  FOREIGN KEY (root_event_id) REFERENCES workflow_events(id) ON DELETE CASCADE,
  FOREIGN KEY (event_id) REFERENCES workflow_events(id) ON DELETE CASCADE,
  FOREIGN KEY (previous_execution_id) REFERENCES workflow_node_executions(id) ON DELETE CASCADE,
  FOREIGN KEY (parent_execution_id) REFERENCES workflow_node_executions(id) ON DELETE CASCADE
);

CREATE INDEX idx_workflows_organization_id ON workflows(organization_id);
CREATE INDEX idx_workflow_events_workflow_node_id ON workflow_events(workflow_id, node_id);
CREATE INDEX idx_workflow_nodes_state ON workflow_nodes(state);
CREATE INDEX idx_workflow_node_executions_workflow_node_id ON workflow_node_executions(workflow_id, node_id);

commit;
