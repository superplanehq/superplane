BEGIN;

ALTER TABLE users ADD COLUMN token_hash character varying(250);

COMMIT;
