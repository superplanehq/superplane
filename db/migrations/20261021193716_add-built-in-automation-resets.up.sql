BEGIN;

CREATE TABLE built_in_automation_resets (
  template_id character varying(255) NOT NULL,
  account_id uuid REFERENCES accounts(id) ON DELETE SET NULL,
  reset_at timestamp without time zone NOT NULL,
  CONSTRAINT built_in_automation_resets_pkey PRIMARY KEY (template_id)
);

CREATE INDEX index_built_in_automation_resets_on_account_id
  ON built_in_automation_resets (account_id);

COMMIT;
