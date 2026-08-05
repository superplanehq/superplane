BEGIN;

--
-- Dedicated timestamp bumped only on work order lifecycle transitions
-- (`UpdateStatus`). Distinct from `updated_at`, which also bumps on
-- assignee changes; the display-status logic uses this column as a
-- fence so failures from a previous attempt don't stick to a reopened
-- order after a reassign, and a reassign doesn't hide a genuine failure.
--
ALTER TABLE factory_work_orders
  ADD COLUMN state_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

--
-- Backfill from the append-only event log first — it's the most accurate
-- record of when a lifecycle transition actually happened. The order and
-- coarse events we've historically emitted map to lifecycle changes:
--   * `order.status.updated` — every explicit UpdateStatus call.
--   * `order.opened` / `order.closed` — legacy coarse events still fired
--     alongside the modern status.updated events; older rows may only
--     have these.
-- Rows without any of those events (very old orders that predate the
-- events table, or created out-of-band) fall back to `created_at` so we
-- keep the failure fence conservative — better to over-flag a rare
-- historical failure than to silently hide one behind a reassign that
-- happened to bump `updated_at`.
--
UPDATE factory_work_orders o
  SET state_updated_at = COALESCE(
    (
      SELECT MAX(e.created_at)
      FROM factory_work_order_events e
      WHERE e.work_order_id = o.id
        AND e.type IN ('order.status.updated', 'order.opened', 'order.closed')
    ),
    o.created_at
  );

COMMIT;
