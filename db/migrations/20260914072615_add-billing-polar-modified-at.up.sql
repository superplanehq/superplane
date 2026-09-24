ALTER TABLE organization_billing_plans
  ADD COLUMN IF NOT EXISTS polar_modified_at timestamp with time zone;
