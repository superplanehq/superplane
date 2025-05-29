begin;

ALTER TABLE stages DROP COLUMN use;
ALTER TABLE stages ADD COLUMN inputs jsonb NOT NULL DEFAULT '[]';
ALTER TABLE stages ADD COLUMN input_mappings jsonb NOT NULL DEFAULT '[]';

commit;
