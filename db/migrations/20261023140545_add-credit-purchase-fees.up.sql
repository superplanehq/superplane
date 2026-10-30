BEGIN;

ALTER TABLE installation_llm_settings
  ADD COLUMN credit_purchase_fee_bps INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN credit_purchase_fixed_fee_cents BIGINT NOT NULL DEFAULT 0,
  ADD CONSTRAINT installation_llm_settings_credit_purchase_fee_range
    CHECK (credit_purchase_fee_bps >= 0 AND credit_purchase_fee_bps < 10000),
  ADD CONSTRAINT installation_llm_settings_credit_purchase_fixed_fee_non_negative
    CHECK (credit_purchase_fixed_fee_cents >= 0);

COMMIT;
