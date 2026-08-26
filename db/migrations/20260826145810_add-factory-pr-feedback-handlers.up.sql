BEGIN;

--
-- A PR feedback handler is a first-class factory resource, the same way an
-- intake is. The row owns identity (which factory, which source, which
-- canvas). The canvas graph stays the single source of truth for behavior.
--
CREATE TABLE factory_pr_feedback_handlers (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL,
  factory_id      UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
  canvas_id       UUID NOT NULL REFERENCES workflows(id) ON DELETE RESTRICT,
  source          VARCHAR(64) NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_factory_pr_feedback_handlers_canvas_id
  ON factory_pr_feedback_handlers (canvas_id);

CREATE INDEX idx_factory_pr_feedback_handlers_factory_id
  ON factory_pr_feedback_handlers (factory_id);

COMMIT;
