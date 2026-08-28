BEGIN;

ALTER TABLE organization_llm_credit_grants
  DROP CONSTRAINT IF EXISTS organization_llm_credit_grants_kind;

ALTER TABLE organization_llm_credit_grants
  ADD CONSTRAINT organization_llm_credit_grants_kind
    CHECK (kind IN ('welcome', 'admin', 'polar'));

ALTER TABLE organization_llm_credit_grants
  ADD COLUMN IF NOT EXISTS polar_order_id TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_org_llm_credit_grants_polar_order
  ON organization_llm_credit_grants (polar_order_id)
  WHERE polar_order_id IS NOT NULL;

ALTER TABLE organization_llm_settings
  ADD COLUMN IF NOT EXISTS polar_customer_id TEXT;

ALTER TABLE organization_llm_credit_holds
  ADD COLUMN IF NOT EXISTS factory_id UUID;

COMMIT;
