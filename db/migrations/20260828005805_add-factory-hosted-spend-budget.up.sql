BEGIN;

ALTER TABLE factories
  ADD COLUMN IF NOT EXISTS hosted_spend_budget_cents BIGINT;

ALTER TABLE factories
  DROP CONSTRAINT IF EXISTS factories_hosted_spend_budget_non_negative;

ALTER TABLE factories
  ADD CONSTRAINT factories_hosted_spend_budget_non_negative
    CHECK (hosted_spend_budget_cents IS NULL OR hosted_spend_budget_cents >= 0);

COMMIT;
