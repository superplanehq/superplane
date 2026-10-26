ALTER TABLE organization_billing_plans
    ADD COLUMN IF NOT EXISTS past_due_since timestamp with time zone;

ALTER TABLE installation_metadata
    ADD COLUMN IF NOT EXISTS past_due_grace_days integer NOT NULL DEFAULT 0
    CHECK (past_due_grace_days BETWEEN 0 AND 21);
