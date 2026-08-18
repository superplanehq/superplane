BEGIN;

--
-- Pull-request lifecycle timestamps on work-order artifacts. The webhook path
-- (github.onPullRequest -> findWorkOrder -> updateWorkOrderArtifact) already
-- knows when a PR merged or closed; storing the moment is what unlocks the
-- Velocity page ("merged in this period", "waste in this period") without
-- N+1 fan-out through work orders.
--
ALTER TABLE factory_work_order_artifacts
  ADD COLUMN merged_at TIMESTAMPTZ,
  ADD COLUMN closed_at TIMESTAMPTZ;

CREATE INDEX idx_factory_work_order_artifacts_pr_merged_at
  ON factory_work_order_artifacts (factory_id, merged_at DESC)
  WHERE type = 'pr' AND merged_at IS NOT NULL;

CREATE INDEX idx_factory_work_order_artifacts_pr_closed_at
  ON factory_work_order_artifacts (factory_id, closed_at DESC)
  WHERE type = 'pr' AND closed_at IS NOT NULL;

COMMIT;
