ALTER TABLE organization_billing_plans
    ADD COLUMN renewal_seats_scheduled integer CHECK (renewal_seats_scheduled >= 1),
    ADD COLUMN renewal_seats_scheduled_at timestamp with time zone;
