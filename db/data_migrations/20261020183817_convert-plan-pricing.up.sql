BEGIN;

-- Convert after the tier seed, which still uses the original pricing columns.
DO $$
BEGIN
 IF EXISTS (SELECT 1 FROM billing_plans WHERE base_price_cents > 0 AND seat_price_cents > 0) THEN
  RAISE EXCEPTION 'Cannot convert plans with both a flat price and a seat price. Choose one price before migrating.';
 END IF;
END $$;

UPDATE billing_plans
 SET price_cents = CASE WHEN seat_price_cents > 0 THEN seat_price_cents ELSE base_price_cents END,
     per_seat = seat_price_cents > 0;

ALTER TABLE billing_plans DROP COLUMN base_price_cents, DROP COLUMN seat_price_cents;

COMMIT;
