BEGIN;

ALTER TABLE connection_group_field_sets
  ADD COLUMN timeout integer,
  ADD COLUMN timeout_behavior CHARACTER VARYING(64),
  ADD COLUMN result CHARACTER VARYING(64);

COMMIT;
