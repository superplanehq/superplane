CREATE INDEX CONCURRENTLY runner_tasks_organization_active_idx
  ON runner_tasks (organization_id)
  WHERE backend = 'integrated'
    AND state IN ('reserved', 'running');
