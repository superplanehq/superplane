-- Enterprise stays private until custom pricing and credits are agreed.
INSERT INTO billing_plans (
 key, name, category, visibility, base_price_cents, seat_price_cents,
 seat_limit, parallel_task_limit, markup_bps, require_payment_method, features
) VALUES
 (
  'pay_as_you_go', 'Pay-as-you-go', 'paid', 'public', 0, 0,
  10, 5, 1000, true, '["byo_subscription"]'::jsonb
 ),
 (
  'starter', 'Starter', 'paid', 'public', 2000, 0,
  10, NULL, 1000, true, '["byo_subscription"]'::jsonb
 ),
 (
  'team', 'Team', 'paid', 'public', 0, 3000,
  50, NULL, 1000, true, '["byo_subscription", "byo_key", "centralized_billing"]'::jsonb
 ),
 (
  'enterprise', 'Enterprise', 'paid', 'private', 0, 0,
  NULL, NULL, NULL, false,
  '["byo_subscription", "byo_key", "centralized_billing", "self_hosted_app", "priority_support"]'::jsonb
 )
ON CONFLICT (key) DO NOTHING;
