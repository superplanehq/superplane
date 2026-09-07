BEGIN;

--
-- Manual drag-and-drop order for work orders on the Tasks board/list.
-- Scoped per factory (not per lane): lanes are derived from display status
-- at read time, so a single ordered sequence per factory is enough to
-- bucket into lanes and keep Board/List in sync. Lower values sort first;
-- ties fall back to the existing created_at/id order.
--
ALTER TABLE factory_work_orders
  ADD COLUMN position DOUBLE PRECISION;

--
-- Backfill existing rows so the manual order starts out matching today's
-- default order (newest first) instead of every row tying at NULL.
--
WITH ranked AS (
  SELECT id, ROW_NUMBER() OVER (
    PARTITION BY factory_id ORDER BY created_at DESC, id DESC
  ) AS rank
  FROM factory_work_orders
)
UPDATE factory_work_orders
SET position = ranked.rank * 1000
FROM ranked
WHERE factory_work_orders.id = ranked.id;

ALTER TABLE factory_work_orders
  ALTER COLUMN position SET NOT NULL;

CREATE INDEX idx_factory_work_orders_factory_position
  ON factory_work_orders (factory_id, position);

COMMIT;
