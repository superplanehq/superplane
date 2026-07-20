ALTER TABLE workflow_runs
  ADD COLUMN output jsonb NOT NULL DEFAULT '{}',
  ADD COLUMN errors jsonb NOT NULL DEFAULT '[]';

-- Now that we have errors, we don't need the result message column anymore.
ALTER TABLE workflow_runs DROP COLUMN result_message;
