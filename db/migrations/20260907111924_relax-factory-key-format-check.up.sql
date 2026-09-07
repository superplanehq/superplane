BEGIN;

--
-- New and edited workspace keys must now be lowercase (see
-- pkg/models/factory.go). Existing keys created before this change may
-- still be uppercase, so the check constraint stays case-insensitive at
-- the database layer; the application layer is what enforces lowercase
-- for writes going forward.
--
ALTER TABLE factories
  DROP CONSTRAINT factories_key_format_check,
  ADD CONSTRAINT factories_key_format_check
    CHECK (key ~ '^[A-Za-z]{2,5}$');

COMMIT;
