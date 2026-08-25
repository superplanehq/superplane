BEGIN;

CREATE TABLE IF NOT EXISTS organization_llm_credit_holds (
  node_execution_id  UUID PRIMARY KEY,
  organization_id    UUID NOT NULL,
  amount_micros      BIGINT NOT NULL,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT organization_llm_credit_holds_amount_positive CHECK (amount_micros > 0)
);

CREATE INDEX IF NOT EXISTS idx_org_llm_credit_holds_org
  ON organization_llm_credit_holds (organization_id);

COMMIT;
