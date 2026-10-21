UPDATE billing_plans
SET price_cents = 0,
    credit_amount_cents = 0,
    updated_at = now()
WHERE key = 'business_manual';
