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

CREATE TABLE factory_sources (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL,
  factory_id      UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
  name            TEXT NOT NULL,
  integration_id  UUID NOT NULL REFERENCES app_installations(id) ON DELETE RESTRICT,
  configuration   JSONB NOT NULL DEFAULT '{}',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_factory_sources_factory_id ON factory_sources (factory_id);

CREATE TABLE factory_agents (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL,
  factory_id      UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
  name            TEXT NOT NULL,
  description     TEXT NOT NULL DEFAULT '',
  spec            JSONB NOT NULL DEFAULT '{}',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  UNIQUE (factory_id, name)
);

CREATE INDEX idx_factory_agents_factory_id ON factory_agents (factory_id);

CREATE TABLE factory_work_orders (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL,
  factory_id      UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
  title           TEXT NOT NULL,
  description     TEXT NOT NULL DEFAULT '',
  state           VARCHAR(32) NOT NULL DEFAULT 'open',
  result          VARCHAR(32) NOT NULL DEFAULT '',
  source_id       UUID REFERENCES factory_sources(id) ON DELETE SET NULL,
  source_name     TEXT NOT NULL DEFAULT '',
  source_key      TEXT NOT NULL DEFAULT '',
  attributes      JSONB NOT NULL DEFAULT '[]',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_factory_work_orders_factory_state ON factory_work_orders (factory_id, state);

CREATE UNIQUE INDEX idx_factory_work_orders_source_dedup
  ON factory_work_orders (factory_id, source_id, source_key)
  WHERE source_key <> '';

CREATE TABLE factory_work_order_assignees (
  work_order_id UUID NOT NULL REFERENCES factory_work_orders(id) ON DELETE RESTRICT,
  user_id       UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  PRIMARY KEY (work_order_id, user_id)
);

CREATE INDEX idx_factory_work_order_assignees_user_id
  ON factory_work_order_assignees (user_id);

CREATE TABLE factory_work_order_events (
  id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  work_order_id UUID NOT NULL REFERENCES factory_work_orders(id) ON DELETE RESTRICT,
  type          TEXT NOT NULL,
  content       JSONB NOT NULL DEFAULT '{}',
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_factory_work_order_events_order_created
  ON factory_work_order_events (work_order_id, created_at DESC, id DESC);

CREATE TABLE factory_agent_assignments (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL,
  factory_id      UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
  agent_id        UUID NOT NULL REFERENCES factory_agents(id) ON DELETE RESTRICT,
  work_order_id   UUID NOT NULL REFERENCES factory_work_orders(id) ON DELETE RESTRICT,
  instructions    TEXT NOT NULL DEFAULT '',
  state           VARCHAR(32) NOT NULL DEFAULT 'pending',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_factory_agent_assignments_factory_id ON factory_agent_assignments (factory_id);
CREATE INDEX idx_factory_agent_assignments_agent_id ON factory_agent_assignments (agent_id);
CREATE INDEX idx_factory_agent_assignments_work_order_id ON factory_agent_assignments (work_order_id);
CREATE INDEX idx_factory_agent_assignments_pending
  ON factory_agent_assignments (state)
  WHERE state IN ('pending', 'started');

COMMIT;
