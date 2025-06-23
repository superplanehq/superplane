BEGIN;

ALTER TABLE stage_connections RENAME TO connections;
DROP INDEX uix_stage_connections_stage;
ALTER TABLE connections RENAME CONSTRAINT stage_connections_pkey TO connections_pkey;
ALTER TABLE connections RENAME CONSTRAINT stage_connections_stage_id_source_id_key TO connections_target_id_source_id_key;
ALTER TABLE connections DROP CONSTRAINT stage_connections_stage_id_fkey;
ALTER TABLE connections RENAME COLUMN stage_id TO target_id;
ALTER TABLE connections ADD COLUMN target_type CHARACTER VARYING(64) NOT NULL DEFAULT 'stage';

COMMIT;