BEGIN;

ALTER TABLE organization_llm_credit_grants
  DROP CONSTRAINT IF EXISTS organization_llm_credit_grants_kind;

ALTER TABLE organization_llm_credit_grants
  DROP CONSTRAINT IF EXISTS organization_llm_credit_grants_amount_sign;

ALTER TABLE organization_llm_credit_grants
  ADD CONSTRAINT organization_llm_credit_grants_kind
    CHECK (kind IN (
      'welcome',
      'admin',
      'included',
      'topup',
      'topup_refund',
      'trial_adjustment',
      'topup_adjustment',
      'admin_adjustment'
    ));

ALTER TABLE organization_llm_credit_grants
  ADD CONSTRAINT organization_llm_credit_grants_amount_sign
    CHECK (
      (kind = 'topup_refund' AND amount_micros < 0)
      OR
      (kind IN ('trial_adjustment', 'topup_adjustment', 'admin_adjustment') AND amount_micros <> 0)
      OR
      (kind IN ('welcome', 'admin', 'included', 'topup') AND amount_micros > 0)
    );

COMMIT;
