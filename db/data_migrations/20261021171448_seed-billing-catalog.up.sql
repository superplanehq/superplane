BEGIN;

-- All seeded plans stay private until pricing and credits are agreed.
INSERT INTO billing_plans (
 key, name, category, visibility, price_cents, per_seat,
 seat_limit, parallel_task_limit, markup_bps, require_payment_method,
 trial_credit_cents, trial_length_days, features
) VALUES (
 'pay_as_you_go', 'Pay-as-you-go', 'paid', 'private', 0, false,
 10, 5, 1000, true, 1000, 14, '["byo_subscription"]'::jsonb
)
ON CONFLICT (key) DO NOTHING;

INSERT INTO billing_plans (
 key, name, category, visibility, price_cents, per_seat,
 seat_limit, parallel_task_limit, credit_amount_cents, credit_frequency,
 credit_expiration, credit_validity_days, trial_credit_cents,
 trial_length_days, markup_bps, features, require_payment_method
)
SELECT
 'business_manual', 'Business Manual', category, 'private', price_cents, per_seat,
 seat_limit, parallel_task_limit, credit_amount_cents, credit_frequency,
 credit_expiration, credit_validity_days, trial_credit_cents,
 trial_length_days, markup_bps, features, false
FROM billing_plans WHERE key = 'business_legacy'
ON CONFLICT (key) DO NOTHING;

-- Preserve subscription history, credit periods, and balances during reassignment.
UPDATE organization_billing_plans o
SET plan_id = manual.id
FROM billing_plans manual, billing_plans legacy
WHERE manual.key = 'business_manual'
 AND legacy.key = 'business_legacy'
 AND o.plan_id = legacy.id
 AND o.plan = 'business'
 AND o.plan_source = 'admin';

COMMIT;
