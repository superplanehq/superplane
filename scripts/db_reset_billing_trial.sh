#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

# Local-only. Resets SuperPlane billing rows so Subscribe can run again.
# This does not change Polar sandbox customers or subscriptions.

DB_NAME="${1:-superplane_dev}"
export PGPASSWORD=the-cake-is-a-lie

if [[ "$DB_NAME" != "superplane_dev" ]]; then
  echo "db.reset.billing.trial only runs against superplane_dev." >&2
  exit 1
fi

psql -h db -p 5432 -U postgres -d "$DB_NAME" -v ON_ERROR_STOP=1 <<'SQL'
BEGIN;

UPDATE organization_billing_plans
SET
  plan = 'trial',
  plan_source = 'system',
  polar_subscription_id = NULL,
  polar_subscription_status = '',
  current_period_start = NULL,
  current_period_end = NULL,
  trial_started_at = NOW(),
  trial_ends_at = NOW() + INTERVAL '14 days',
  updated_at = NOW();

UPDATE organization_llm_settings
SET
  polar_customer_id = NULL,
  updated_at = NOW()
WHERE polar_customer_id IS NOT NULL;

DELETE FROM organization_llm_credit_grants
WHERE kind IN ('included', 'topup', 'topup_refund');

UPDATE organization_llm_credit_grants
SET expires_at = NOW() + INTERVAL '14 days'
WHERE kind = 'welcome';

COMMIT;

SELECT
  COUNT(*) AS organizations_on_trial
FROM organization_billing_plans
WHERE plan = 'trial' AND plan_source = 'system';
SQL
