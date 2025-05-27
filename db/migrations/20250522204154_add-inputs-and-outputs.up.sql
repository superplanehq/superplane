begin;

ALTER TABLE stages DROP COLUMN use;
ALTER TABLE stage_executions RENAME COLUMN tags TO outputs;

commit;