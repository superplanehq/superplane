-- Before dispatch markers existed, a registered runner could receive its
-- payload before the reserved-to-running transaction committed. Preserve that
-- uncertainty on upgrade. Pending runners have never received a payload.
UPDATE runner_tasks AS t
SET dispatch_started_at = COALESCE(t.reserved_at, t.created_at)
FROM runners AS r
WHERE t.runner_id = r.id
  AND t.backend = 'integrated'
  AND t.state = 'reserved'
  AND t.dispatch_started_at IS NULL
  AND r.state <> 'pending';
