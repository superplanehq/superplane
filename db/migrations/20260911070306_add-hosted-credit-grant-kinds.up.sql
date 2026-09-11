BEGIN;

DROP INDEX IF EXISTS idx_org_llm_credit_grants_polar_order;

ALTER TABLE organization_llm_credit_grants
  DROP CONSTRAINT IF EXISTS organization_llm_credit_grants_kind;

ALTER TABLE organization_llm_credit_grants
  DROP CONSTRAINT IF EXISTS organization_llm_credit_grants_amount_sign;

UPDATE organization_llm_credit_grants
SET
  kind = 'topup',
  expires_at = COALESCE(expires_at, created_at + INTERVAL '12 months')
WHERE kind = 'polar';

UPDATE organization_llm_credit_grants
SET kind = 'topup_refund'
WHERE kind = 'polar_refund';

ALTER TABLE organization_llm_credit_grants
  ADD CONSTRAINT organization_llm_credit_grants_kind
    CHECK (kind IN ('welcome', 'admin', 'included', 'topup', 'topup_refund'));

ALTER TABLE organization_llm_credit_grants
  ADD CONSTRAINT organization_llm_credit_grants_amount_sign
    CHECK (
      (kind = 'topup_refund' AND amount_micros < 0)
      OR
      (kind <> 'topup_refund' AND amount_micros > 0)
    );

CREATE UNIQUE INDEX IF NOT EXISTS idx_org_llm_credit_grants_polar_order
  ON organization_llm_credit_grants (polar_order_id)
  WHERE polar_order_id IS NOT NULL AND kind IN ('topup', 'included');

COMMIT;
