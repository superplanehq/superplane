BEGIN;

--
-- An intake becomes a first-class row instead of being inferred by scanning a
-- factory canvas for a known trigger, an analysis node, and a createWorkOrder
-- node. Detection could not express more than one intake per source, and it
-- dropped an intake out of the UI whenever somebody renamed a node or swapped
-- the agent runner.
--
-- The row owns identity only (which factory, which source, which canvas). The
-- canvas graph stays the single source of truth for behavior — thresholds and
-- filters must live in `canvas.yaml` because that is what the workers execute.
--
CREATE TABLE factory_intakes (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL,
  factory_id      UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
  canvas_id       UUID NOT NULL REFERENCES workflows(id) ON DELETE RESTRICT,
  source          VARCHAR(64) NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- A canvas implements at most one intake, so the canvas reference is the
-- natural key. This also makes the "is this canvas an intake" lookup that the
-- app list performs a single indexed probe.
CREATE UNIQUE INDEX idx_factory_intakes_canvas_id
  ON factory_intakes (canvas_id);

CREATE INDEX idx_factory_intakes_factory_id
  ON factory_intakes (factory_id);

COMMIT;
