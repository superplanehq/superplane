BEGIN;

-- IF NOT EXISTS lets databases that already created this table under an
-- older timestamp (20260819005114, 20260819020000, or 20260819142559)
-- apply this version without failing.

CREATE TABLE IF NOT EXISTS llm_usage_events (
  id                       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id          UUID NOT NULL,
  factory_id               UUID,
  work_order_id            UUID,
  line_id                  UUID,
  line_dispatch_id         UUID,
  work_order_execution_id  UUID,
  canvas_run_id            UUID NOT NULL,
  node_execution_id        UUID NOT NULL,
  node_id                  TEXT NOT NULL,
  provider                 TEXT NOT NULL,
  model                    TEXT NOT NULL,
  usage_kind               TEXT NOT NULL DEFAULT 'model',
  funding_source           TEXT NOT NULL DEFAULT 'byok',
  input_tokens             BIGINT NOT NULL DEFAULT 0,
  output_tokens            BIGINT NOT NULL DEFAULT 0,
  cache_read_tokens        BIGINT NOT NULL DEFAULT 0,
  cache_write_tokens       BIGINT NOT NULL DEFAULT 0,
  reasoning_tokens         BIGINT NOT NULL DEFAULT 0,
  total_tokens             BIGINT NOT NULL DEFAULT 0,
  cost_micros              BIGINT NOT NULL DEFAULT 0,
  currency                 TEXT NOT NULL DEFAULT 'usd',
  price_book_version       TEXT NOT NULL,
  idempotency_key          TEXT NOT NULL,
  occurred_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  UNIQUE (idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_llm_usage_events_org_occurred
  ON llm_usage_events (organization_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_llm_usage_events_factory_occurred
  ON llm_usage_events (factory_id, occurred_at DESC)
  WHERE factory_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_llm_usage_events_work_order
  ON llm_usage_events (work_order_id)
  WHERE work_order_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_llm_usage_events_execution
  ON llm_usage_events (work_order_execution_id)
  WHERE work_order_execution_id IS NOT NULL;

COMMIT;
