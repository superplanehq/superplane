begin;

ALTER TABLE stages DROP COLUMN use;
ALTER TABLE stage_executions RENAME COLUMN tags TO outputs;
ALTER TABLE stage_events ADD COLUMN inputs jsonb NOT NULL DEFAULT '{}';

commit;