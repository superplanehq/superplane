BEGIN;

--
-- Aggregate usage per execution (tokens, cost). Populated later by runners;
-- the API surfaces both per-execution values and a summed total on the
-- containing work order so the sidebar can render "528k tokens - $1.85".
--
ALTER TABLE factory_work_order_executions
  ADD COLUMN total_tokens BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN cost_cents BIGINT NOT NULL DEFAULT 0;

COMMIT;
