begin;

ALTER TABLE stages ADD COLUMN dry_run boolean DEFAULT false;

commit;