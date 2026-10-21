ALTER TABLE billing_plans
 ADD COLUMN price_cents bigint NOT NULL DEFAULT 0 CHECK (price_cents >= 0),
 ADD COLUMN per_seat boolean NOT NULL DEFAULT false;
