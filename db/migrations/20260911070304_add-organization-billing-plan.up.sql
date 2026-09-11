BEGIN;

CREATE TABLE IF NOT EXISTS organization_billing_plans (
  organization_id            UUID PRIMARY KEY,
  plan                      TEXT NOT NULL,
  plan_source               TEXT NOT NULL DEFAULT '',
  polar_subscription_id      TEXT,
  polar_subscription_status  TEXT NOT NULL DEFAULT '',
  current_period_start      TIMESTAMPTZ,
  current_period_end        TIMESTAMPTZ,
  trial_started_at          TIMESTAMPTZ,
  trial_ends_at            TIMESTAMPTZ,
  updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT organization_billing_plans_plan CHECK (plan IN ('trial', 'business', 'none')),
  CONSTRAINT organization_billing_plans_source CHECK (plan_source IN ('', 'system', 'polar', 'admin'))
);

INSERT INTO organization_billing_plans (
  organization_id,
  plan,
  plan_source,
  trial_started_at,
  trial_ends_at,
  updated_at
)
SELECT
  id,
  'trial',
  'system',
  NOW(),
  NOW() + INTERVAL '14 days',
  NOW()
FROM organizations
WHERE deleted_at IS NULL
ON CONFLICT (organization_id) DO NOTHING;

COMMIT;
