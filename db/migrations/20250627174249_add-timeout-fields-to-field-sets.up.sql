BEGIN;

ALTER TABLE connection_group_field_sets
  ADD COLUMN timeout integer,
  ADD COLUMN timeout_behavior CHARACTER VARYING(64),
  ADD COLUMN state_reason CHARACTER VARYING(64);

COMMIT;
