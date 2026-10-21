BEGIN;

-- Convert after the tier seed, which still uses the original pricing columns.
UPDATE billing_plans
 SET price_cents = base_price_cents,
     per_seat = false;

ALTER TABLE billing_plans DROP COLUMN base_price_cents, DROP COLUMN seat_price_cents;

COMMIT;
