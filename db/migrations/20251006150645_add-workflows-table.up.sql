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
-- Workflow initial events table and indexes
-- Stores the initial trigger data for workflows
--

CREATE TABLE workflow_initial_events (
  id          uuid NOT NULL DEFAULT uuid_generate_v4(),
  workflow_id uuid NOT NULL,
  data        JSONB NOT NULL,
  created_at  TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE
);

CREATE INDEX idx_workflow_initial_events_workflow_id ON workflow_initial_events(workflow_id);

--
-- Workflows node executions table and indexes
--

CREATE TABLE workflow_node_executions (
  id                     uuid NOT NULL DEFAULT uuid_generate_v4(),
  workflow_id            uuid NOT NULL,
  node_id                CHARACTER VARYING(128) NOT NULL,

  -- Root event (shared by all executions in this workflow run)
  root_event_id          uuid NOT NULL,

  -- Sequential flow (previous node that provides inputs)
  previous_execution_id  uuid,
  previous_output_branch CHARACTER VARYING(64),
  previous_output_index  INTEGER,

  -- Blueprint hierarchy (blueprint node execution that spawned this, if any)
  parent_execution_id    uuid,

  -- Blueprint context (no FK - we want to preserve execution history even after blueprint deletion)
  blueprint_id           uuid,

  -- State machine
  state                  CHARACTER VARYING(32) NOT NULL,
  result                 CHARACTER VARYING(32),
  result_reason          CHARACTER VARYING(128),
  result_message         TEXT,

  -- Data (only outputs stored, inputs derived from previous)
  outputs                JSONB,

  -- Node snapshot
  metadata               JSONB NOT NULL DEFAULT '{}'::jsonb,
  configuration          JSONB NOT NULL DEFAULT '{}'::jsonb,

  created_at             TIMESTAMP NOT NULL,
  updated_at             TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE,
  FOREIGN KEY (root_event_id) REFERENCES workflow_initial_events(id) ON DELETE CASCADE,
  FOREIGN KEY (previous_execution_id) REFERENCES workflow_node_executions(id) ON DELETE CASCADE,
  FOREIGN KEY (parent_execution_id) REFERENCES workflow_node_executions(id) ON DELETE CASCADE
);

CREATE INDEX idx_workflow_node_executions_workflow_id ON workflow_node_executions(workflow_id, node_id);
CREATE INDEX idx_workflow_node_executions_root_event ON workflow_node_executions(root_event_id);
CREATE INDEX idx_workflow_node_executions_previous ON workflow_node_executions(previous_execution_id);
CREATE INDEX idx_workflow_node_executions_parent ON workflow_node_executions(parent_execution_id);
CREATE INDEX idx_workflow_node_executions_blueprint ON workflow_node_executions(blueprint_id) WHERE blueprint_id IS NOT NULL;
CREATE INDEX idx_workflow_node_executions_state_pending ON workflow_node_executions(state) WHERE state = 'pending';
CREATE INDEX idx_workflow_node_executions_state_routing ON workflow_node_executions(state) WHERE state = 'routing';

commit;
