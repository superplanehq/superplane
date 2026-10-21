-- All seeded plans stay private until pricing and credits are agreed.
INSERT INTO billing_plans (
 key, name, category, visibility, base_price_cents, seat_price_cents,
 seat_limit, parallel_task_limit, markup_bps, require_payment_method, features
) VALUES
 (
  'pay_as_you_go', 'Pay-as-you-go', 'paid', 'private', 0, 0,
  10, 5, 1000, true, '["byo_subscription"]'::jsonb
 )
ON CONFLICT (key) DO NOTHING;
