BEGIN;

ALTER TABLE organization_llm_credit_grants
  DROP CONSTRAINT IF EXISTS organization_llm_credit_grants_kind;

ALTER TABLE organization_llm_credit_grants
  ADD CONSTRAINT organization_llm_credit_grants_kind
    CHECK (kind IN ('welcome', 'admin', 'polar', 'polar_refund'));

ALTER TABLE organization_llm_credit_grants
  DROP CONSTRAINT IF EXISTS organization_llm_credit_grants_amount_positive;

ALTER TABLE organization_llm_credit_grants
  ADD CONSTRAINT organization_llm_credit_grants_amount_sign
    CHECK (
      (kind = 'polar_refund' AND amount_micros < 0)
      OR
      (kind <> 'polar_refund' AND amount_micros > 0)
    );

ALTER TABLE organization_llm_credit_grants
  ADD COLUMN IF NOT EXISTS polar_refund_id TEXT;

DROP INDEX IF EXISTS idx_org_llm_credit_grants_polar_order;

CREATE UNIQUE INDEX IF NOT EXISTS idx_org_llm_credit_grants_polar_order
  ON organization_llm_credit_grants (polar_order_id)
  WHERE polar_order_id IS NOT NULL AND kind = 'polar';

CREATE UNIQUE INDEX IF NOT EXISTS idx_org_llm_credit_grants_polar_refund
  ON organization_llm_credit_grants (polar_refund_id)
  WHERE polar_refund_id IS NOT NULL;

COMMIT;
