BEGIN;

ALTER TABLE files
  ADD COLUMN purpose character varying(32) NOT NULL DEFAULT 'attachment',
  ADD COLUMN public_id uuid;

ALTER TABLE files
  ADD CONSTRAINT files_purpose_check
  CHECK (purpose IN ('attachment', 'artifact'));

CREATE UNIQUE INDEX files_public_id_unique
  ON files (public_id)
  WHERE public_id IS NOT NULL;

COMMIT;
