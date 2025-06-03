begin;

ALTER TABLE stages DROP COLUMN use;
ALTER TABLE stages ADD COLUMN inputs jsonb NOT NULL DEFAULT '[]';
ALTER TABLE stages ADD COLUMN outputs jsonb NOT NULL DEFAULT '[]';
ALTER TABLE stages ADD COLUMN input_mappings jsonb NOT NULL DEFAULT '[]';
ALTER TABLE stage_events ADD COLUMN inputs jsonb NOT NULL DEFAULT '{}';
ALTER TABLE stage_executions RENAME COLUMN tags TO outputs;
ALTER TABLE stage_executions ALTER COLUMN outputs SET DEFAULT '{}';
ALTER TABLE stage_executions ALTER COLUMN outputs SET NOT NULL;

commit;
