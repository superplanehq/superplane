-- Keep the backfill and SET NOT NULL in one lock so a concurrent insert
-- cannot land a NULL url_id after the backfill. golang-migrate wraps this
-- file in one transaction; ACCESS EXCLUSIVE blocks writers until commit.
LOCK TABLE factories IN ACCESS EXCLUSIVE MODE;

ALTER TABLE factories ADD COLUMN url_id TEXT;

DO $$
DECLARE
  factory_row RECORD;
  candidate TEXT;
  alphabet TEXT := 'abcdefghijklmnopqrstuvwxyz0123456789';
  i INT;
BEGIN
  FOR factory_row IN
    SELECT id FROM factories WHERE url_id IS NULL ORDER BY created_at ASC, id ASC
  LOOP
    LOOP
      candidate := '';
      FOR i IN 1..8 LOOP
        candidate := candidate || substr(alphabet, 1 + floor(random() * 36)::int, 1);
      END LOOP;
      EXIT WHEN NOT EXISTS (SELECT 1 FROM factories WHERE url_id = candidate);
    END LOOP;
    UPDATE factories SET url_id = candidate WHERE id = factory_row.id;
  END LOOP;
END $$;

ALTER TABLE factories
  ALTER COLUMN url_id SET NOT NULL,
  ADD CONSTRAINT factories_url_id_format_check
    CHECK (url_id ~ '^[a-z0-9]{8}$');

CREATE UNIQUE INDEX factories_url_id_key ON factories (url_id);
