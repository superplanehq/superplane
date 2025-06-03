begin;

ALTER TABLE stages RENAME COLUMN run_template TO executor_spec;

commit;
