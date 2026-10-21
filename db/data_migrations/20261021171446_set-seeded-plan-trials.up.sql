UPDATE billing_plans
SET trial_credit_cents = 1000,
    trial_length_days = 14
WHERE key = 'pay_as_you_go';
