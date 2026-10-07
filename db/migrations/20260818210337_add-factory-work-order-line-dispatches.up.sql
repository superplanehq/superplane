BEGIN;

--
-- The traversal of a work order through a factory line becomes a first-class
-- row instead of being implied by (work_order_id, line_id) grouping over
-- factory_work_order_executions. Step executions become children of a
-- FactoryWorkOrderLineDispatch, referencing it via line_dispatch_id. See
-- issue #6737.
--
CREATE TABLE factory_work_order_line_dispatches (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL,
  factory_id      UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
  work_order_id   UUID NOT NULL REFERENCES factory_work_orders(id) ON DELETE RESTRICT,
  line_id         UUID NOT NULL REFERENCES factory_lines(id) ON DELETE RESTRICT,
  line_name       TEXT NOT NULL,
  -- Snapshot of factory_lines.steps taken at dispatch time. Advancement
  -- reads this, not the live line, so a mid-traversal line edit can't
  -- change which step runs next for an in-flight traversal.
  steps           JSONB NOT NULL DEFAULT '[]'::jsonb,
  state           VARCHAR(32) NOT NULL DEFAULT 'active',
  result          VARCHAR(32) NOT NULL DEFAULT '',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  finished_at     TIMESTAMPTZ
);

-- The dispatch guard and the open -> draft revert guard both ask "does this
-- work order have an active traversal right now" — a single indexed lookup,
-- replacing a scan over child executions by status.
CREATE INDEX idx_factory_work_order_line_dispatches_active
  ON factory_work_order_line_dispatches (work_order_id)
  WHERE state = 'active';

CREATE INDEX idx_factory_work_order_line_dispatches_work_order
  ON factory_work_order_line_dispatches (work_order_id);

CREATE INDEX idx_factory_work_order_line_dispatches_factory_id
  ON factory_work_order_line_dispatches (factory_id);

--
-- factory_work_order_executions gains a required parent reference. Added
-- nullable first so the backfill below can populate existing rows before
-- the NOT NULL constraint is enforced.
--
ALTER TABLE factory_work_order_executions
  ADD COLUMN line_dispatch_id UUID;

--
-- Backfill: synthesize one line dispatch per (work_order_id, line_id) group
-- of existing executions.
--
-- - steps/line_name come from the *current* factory_lines row — the
--   dispatch-time value can't be recovered, this is accepted lossy
--   backfill behavior.
-- - state is `active` if any child execution is still pending/running,
--   otherwise `finished`, with the result of the group's most recently
--   finished child (falling back to the most recently updated child when
--   finished_at is null), matching the "latest execution wins" semantics
--   the API/UI use today.
-- - created_at is the group's earliest execution, finished_at the group's
--   latest, when finished.
-- - Historical re-dispatches of the same line collapse into one
--   synthesized traversal — the same fidelity the UI shows today, so
--   nothing is lost (see issue #6737).
--
WITH groups AS (
  SELECT
    e.work_order_id,
    e.line_id,
    MIN(e.created_at)  AS created_at,
    MAX(e.finished_at) AS finished_at,
    BOOL_OR(e.status IN ('pending', 'running')) AS has_active
  FROM factory_work_order_executions e
  GROUP BY e.work_order_id, e.line_id
),
latest_result AS (
  SELECT DISTINCT ON (e.work_order_id, e.line_id)
    e.work_order_id,
    e.line_id,
    e.result AS latest_result
  FROM factory_work_order_executions e
  ORDER BY
    e.work_order_id,
    e.line_id,
    e.finished_at DESC NULLS LAST,
    e.updated_at DESC,
    e.id DESC
),
inserted AS (
  INSERT INTO factory_work_order_line_dispatches (
    organization_id, factory_id, work_order_id, line_id, line_name, steps,
    state, result, created_at, updated_at, finished_at
  )
  SELECT
    wo.organization_id,
    wo.factory_id,
    g.work_order_id,
    g.line_id,
    COALESCE(l.name, ''),
    COALESCE(l.steps, '[]'::jsonb),
    CASE WHEN g.has_active THEN 'active' ELSE 'finished' END,
    CASE WHEN g.has_active THEN '' ELSE COALESCE(lr.latest_result, '') END,
    g.created_at,
    NOW(),
    CASE WHEN g.has_active THEN NULL ELSE g.finished_at END
  FROM groups g
  JOIN factory_work_orders wo ON wo.id = g.work_order_id
  LEFT JOIN factory_lines l ON l.id = g.line_id
  LEFT JOIN latest_result lr
    ON lr.work_order_id = g.work_order_id AND lr.line_id = g.line_id
  RETURNING id, work_order_id, line_id
)
UPDATE factory_work_order_executions e
SET line_dispatch_id = inserted.id
FROM inserted
WHERE e.work_order_id = inserted.work_order_id
  AND e.line_id = inserted.line_id;

ALTER TABLE factory_work_order_executions
  ALTER COLUMN line_dispatch_id SET NOT NULL;

ALTER TABLE factory_work_order_executions
  ADD FOREIGN KEY (line_dispatch_id)
  REFERENCES factory_work_order_line_dispatches(id) ON DELETE RESTRICT;

CREATE INDEX idx_factory_work_order_executions_line_dispatch
  ON factory_work_order_executions (line_dispatch_id);

COMMIT;
