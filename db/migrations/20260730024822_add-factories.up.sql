BEGIN;

CREATE TABLE factories (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL,
  name            TEXT NOT NULL,
  description     TEXT NOT NULL DEFAULT '',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  UNIQUE (organization_id, name)
);

CREATE INDEX idx_factories_organization_id ON factories (organization_id);

CREATE TABLE factory_work_orders (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL,
  factory_id      UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
  title           TEXT NOT NULL,
  description     TEXT NOT NULL DEFAULT '',
  state           VARCHAR(32) NOT NULL DEFAULT 'open',
  result          VARCHAR(32) NOT NULL DEFAULT '',
  created_by_id   UUID REFERENCES users(id) ON DELETE RESTRICT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_factory_work_orders_factory_state ON factory_work_orders (factory_id, state);

CREATE TABLE factory_work_order_assignees (
  work_order_id UUID NOT NULL REFERENCES factory_work_orders(id) ON DELETE RESTRICT,
  user_id       UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  PRIMARY KEY (work_order_id, user_id)
);

CREATE INDEX idx_factory_work_order_assignees_user_id
  ON factory_work_order_assignees (user_id);

--
-- Factory-owned apps (workflows / canvases with factory_id set).
--
ALTER TABLE workflows
  ADD COLUMN factory_id UUID REFERENCES factories(id) ON DELETE RESTRICT;

CREATE INDEX idx_workflows_factory_id ON workflows (factory_id)
  WHERE factory_id IS NOT NULL;

--
-- Factory lines: named pipelines of runApp steps stored as JSON.
-- Each step: { "name", "type": "runApp", "app_id", "entrypoint" }.
--
CREATE TABLE factory_lines (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL,
  factory_id      UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
  name            TEXT NOT NULL,
  steps           JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  UNIQUE (factory_id, name)
);

CREATE INDEX idx_factory_lines_factory_id ON factory_lines (factory_id);

--
-- Tracks a work order progressing through a factory line, one row per step run.
--
CREATE TABLE factory_work_order_executions (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL,
  factory_id      UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
  work_order_id   UUID NOT NULL REFERENCES factory_work_orders(id) ON DELETE RESTRICT,
  line_id         UUID NOT NULL REFERENCES factory_lines(id) ON DELETE RESTRICT,
  step_index      INT NOT NULL,
  step_name       TEXT NOT NULL,
  run_id          UUID NOT NULL REFERENCES workflow_runs(id) ON DELETE RESTRICT,
  status          VARCHAR(32) NOT NULL DEFAULT 'pending',
  result          VARCHAR(32) NOT NULL DEFAULT '',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  finished_at     TIMESTAMPTZ,

  UNIQUE (run_id)
);

CREATE INDEX idx_factory_work_order_executions_work_order_created
  ON factory_work_order_executions (work_order_id, created_at DESC);

CREATE INDEX idx_factory_work_order_executions_work_order_line_active
  ON factory_work_order_executions (work_order_id, line_id)
  WHERE status IN ('pending', 'running');

COMMIT;
