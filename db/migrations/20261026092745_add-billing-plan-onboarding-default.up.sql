ALTER TABLE billing_plans
  ADD COLUMN onboarding_default boolean NOT NULL DEFAULT false;

-- At most one plan is the default after the onboarding card step.
CREATE UNIQUE INDEX billing_plans_single_onboarding_default
  ON billing_plans ((true))
  WHERE onboarding_default;
