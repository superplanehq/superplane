#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

DB_NAME="${1:-superplane_dev}"

if [[ "$DB_NAME" != "superplane_dev" || "${DB_HOST:-db}" != "db" ]]; then
  echo "Local billing catalog setup only runs against superplane_dev on the Docker db service." >&2
  exit 1
fi

if [[ "${POLAR_ENVIRONMENT:-sandbox}" != "sandbox" ||
      "${POLAR_API_BASE_URL:-https://sandbox-api.polar.sh/v1}" != "https://sandbox-api.polar.sh/v1" ]]; then
  echo "Local billing catalog setup requires the Polar sandbox." >&2
  exit 1
fi

export PGPASSWORD="${DB_PASSWORD:-the-cake-is-a-lie}"

psql \
  -X \
  -v ON_ERROR_STOP=1 \
  -h db \
  -p "${DB_PORT:-5432}" \
  -U "${DB_USERNAME:-postgres}" \
  "$DB_NAME" <<'SQL'
BEGIN;

-- These product IDs and prices belong to the Polar sandbox.
INSERT INTO billing_plans (
  key, name, category, visibility, price_cents, per_seat,
  polar_product_id, seat_limit
)
VALUES
  (
    'team', 'Team', 'paid', 'public', 5000, true,
    '633e8026-988b-44de-8d65-31829948c975', 50
  ),
  (
    'starter', 'Starter', 'paid', 'public', 2000, false,
    'abc9ef98-20c0-48b5-8204-1448df733737', 10
  )
ON CONFLICT (key) DO NOTHING;

-- Pay-as-you-go has no price and no Polar product. New organizations start on
-- it after the onboarding card step. Signup credit is independent of the plan.
INSERT INTO billing_plans (
  key, name, category, visibility, price_cents, per_seat,
  trial_credit_cents, trial_length_days
)
VALUES ('pay_as_you_go', 'Pay-as-you-go', 'paid', 'public', 0, false, 0, 365)
ON CONFLICT (key) DO NOTHING;

-- Starter plans share signup credit and differ only in task concurrency.
UPDATE billing_plans
SET seat_limit = 10,
    parallel_task_limit = CASE WHEN key = 'pay_as_you_go' THEN 5 ELSE NULL END,
    trial_credit_cents = 0,
    updated_at = NOW()
WHERE key IN ('pay_as_you_go', 'starter');

-- Enforce plan access locally. BYOS remains available through its plan feature.
UPDATE plan_feature_definitions
SET gated = true
WHERE key <> 'byo_subscription' AND NOT gated;

WITH policies AS (
  SELECT key,
    CASE WHEN key = 'team'
      THEN (SELECT COALESCE(jsonb_agg(key ORDER BY key), '[]'::jsonb) FROM plan_feature_definitions)
      ELSE '["byo_subscription"]'::jsonb
    END AS features
  FROM billing_plans
  WHERE key IN ('team', 'starter', 'pay_as_you_go')
)
UPDATE billing_plans AS plan
SET visibility = 'public',
    require_payment_method = true,
    features = policies.features,
    updated_at = NOW()
FROM policies
WHERE plan.key = policies.key
  AND (plan.visibility <> 'public'
    OR NOT plan.require_payment_method
    OR plan.features IS DISTINCT FROM policies.features);

COMMIT;
SQL
