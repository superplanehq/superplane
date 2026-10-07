BEGIN;

--
-- Add a Jira-style workspace key (e.g. `SP`) and factory-scoped work order
-- numbering (e.g. `SP-1`). UUIDs stay authoritative for routes and joins;
-- (key, number) is a display-only pair that users see across the UI, CLI,
-- and REST responses.
--
ALTER TABLE factories
  ADD COLUMN key                    VARCHAR(5),
  ADD COLUMN next_work_order_number BIGINT NOT NULL DEFAULT 1;

ALTER TABLE factory_work_orders
  ADD COLUMN number BIGINT;

--
-- Backfill workspace keys from factory names for every existing factory.
-- Keys must match `^[A-Z]{2,5}$` per the check constraint below, so we
-- resolve collisions by cycling the last character through the alphabet
-- (KEY, KEB, KEC, ...). Deleted factories share the same key space so a
-- resurrected key never surprises a live workspace later.
--
DO $$
DECLARE
  factory_row RECORD;
  seed        TEXT;
  candidate   TEXT;
  suffix_char INT;
BEGIN
  FOR factory_row IN
    SELECT id, organization_id, name
    FROM factories
    ORDER BY created_at ASC, id ASC
  LOOP
    seed := regexp_replace(upper(coalesce(factory_row.name, '')), '[^A-Z]', '', 'g');
    IF length(seed) = 0 THEN
      seed := 'WS';
    END IF;

    candidate := left(seed, 5);
    IF length(candidate) < 2 THEN
      candidate := rpad(candidate, 2, 'X');
    END IF;

    suffix_char := 0;
    WHILE EXISTS (
      SELECT 1 FROM factories
      WHERE organization_id = factory_row.organization_id
        AND key = candidate
    ) LOOP
      candidate := left(candidate, length(candidate) - 1) || chr(ASCII('A') + suffix_char);
      suffix_char := suffix_char + 1;
      IF suffix_char >= 26 THEN
        -- Extremely unlikely: fall back to WS + counter overflow letters.
        candidate := 'WS' || chr(ASCII('A') + (suffix_char % 26));
        suffix_char := suffix_char + 1;
      END IF;
    END LOOP;

    UPDATE factories SET key = candidate WHERE id = factory_row.id;
  END LOOP;
END$$;

--
-- Backfill sequence numbers per factory in creation order. next value is
-- max(number)+1 so future INSERTs pick up where the backfill left off.
--
WITH numbered AS (
  SELECT id, ROW_NUMBER() OVER (
    PARTITION BY factory_id ORDER BY created_at ASC, id ASC
  ) AS seq
  FROM factory_work_orders
)
UPDATE factory_work_orders
SET number = numbered.seq
FROM numbered
WHERE factory_work_orders.id = numbered.id;

UPDATE factories f
SET next_work_order_number = COALESCE((
  SELECT MAX(number) + 1 FROM factory_work_orders WHERE factory_id = f.id
), 1);

--
-- Lock everything down now that data is consistent.
--
ALTER TABLE factories
  ALTER COLUMN key SET NOT NULL,
  ADD CONSTRAINT factories_key_format_check
    CHECK (key ~ '^[A-Z]{2,5}$');

-- Unique among live factories only. Soft-deleted factories keep their key
-- for history but no longer block a new workspace from reusing it once
-- the constraint is scoped to `deleted_at IS NULL`.
CREATE UNIQUE INDEX factories_organization_id_key_active_key
  ON factories (organization_id, key)
  WHERE deleted_at IS NULL;

ALTER TABLE factory_work_orders
  ALTER COLUMN number SET NOT NULL,
  ADD CONSTRAINT factory_work_orders_number_positive_check
    CHECK (number > 0);

CREATE UNIQUE INDEX factory_work_orders_factory_id_number_key
  ON factory_work_orders (factory_id, number);

COMMIT;
