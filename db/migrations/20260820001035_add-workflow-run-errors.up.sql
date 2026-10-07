ALTER TABLE workflow_runs
  ADD COLUMN errors jsonb NOT NULL DEFAULT '[]';

UPDATE workflow_runs
SET errors = jsonb_build_array(jsonb_build_object('message', result_message))
WHERE result_message IS NOT NULL
  AND btrim(result_message) <> '';

ALTER TABLE workflow_runs DROP COLUMN result_message;
