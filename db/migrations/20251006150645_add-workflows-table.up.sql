begin;

--
-- Workflows table and indexes
--

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

--
-- Workflows events table and indexes
--

CREATE TABLE workflow_events (
  id                 uuid NOT NULL DEFAULT uuid_generate_v4(),
  workflow_id        uuid NOT NULL,
  parent_event_id    uuid,
  blueprint_name     CHARACTER VARYING(128),
  data               JSONB NOT NULL,
  state              CHARACTER VARYING(32) NOT NULL,
  created_at         TIMESTAMP NOT NULL,
  updated_at         TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE,
  FOREIGN KEY (parent_event_id) REFERENCES workflow_events(id) ON DELETE CASCADE
);

CREATE INDEX idx_workflow_events_workflow_id ON workflow_events(workflow_id);
CREATE INDEX idx_workflow_events_parent_event_id ON workflow_events(parent_event_id);
CREATE INDEX idx_workflow_events_state ON workflow_events(state);

--
-- Workflows queue items table and indexes
--

CREATE TABLE workflow_queue_items (
  workflow_id uuid NOT NULL,
  node_id     CHARACTER VARYING(128) NOT NULL,
  event_id    uuid NOT NULL,
  created_at  TIMESTAMP NOT NULL,

  PRIMARY KEY (workflow_id, node_id, event_id),
  FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE,
  FOREIGN KEY (event_id) REFERENCES workflow_events(id) ON DELETE CASCADE
);

CREATE INDEX idx_workflow_queue_items_workflow_node_id ON workflow_queue_items(workflow_id, node_id);

--
-- Workflows node executions table and indexes
--

CREATE TABLE workflow_node_executions (
  id              uuid NOT NULL DEFAULT uuid_generate_v4(),
  event_id        uuid NOT NULL,
  workflow_id     uuid NOT NULL,
  node_id         CHARACTER VARYING(128) NOT NULL,
  state           CHARACTER VARYING(32) NOT NULL,
  result          CHARACTER VARYING(32),
  result_reason   CHARACTER VARYING(128),
  result_message  TEXT,
  inputs          JSONB,
  outputs         JSONB,
  metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
  configuration   JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at      TIMESTAMP NOT NULL,
  updated_at      TIMESTAMP NOT NULL,

  PRIMARY KEY (workflow_id, node_id, event_id),
  FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE,
  FOREIGN KEY (event_id) REFERENCES workflow_events(id) ON DELETE CASCADE
);

CREATE INDEX idx_workflow_node_executions_workflow_id ON workflow_node_executions(workflow_id, node_id);

commit;
