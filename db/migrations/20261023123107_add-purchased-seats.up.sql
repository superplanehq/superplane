ALTER TABLE organization_billing_plans
    ADD COLUMN purchased_seats integer
    CHECK (purchased_seats >= 0);
