BEGIN;

ALTER TABLE secrets
  DROP COLUMN canvas_id,
  ADD COLUMN domain_type CHARACTER VARYING(64) NOT NULL,
  ADD COLUMN domain_id CHARACTER VARYING(64) NOT NULL,
  ADD CONSTRAINT secrets_domain_id_name_key UNIQUE (domain_type, domain_id, name);

COMMIT;
