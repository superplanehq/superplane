BEGIN;

--
-- Restores the uppercase-only constraint. This fails if any row was
-- written with a lowercase key while the relaxed constraint was active —
-- resolve those rows (or delete them) before rolling back.
--
ALTER TABLE factories
  DROP CONSTRAINT factories_key_format_check,
  ADD CONSTRAINT factories_key_format_check
    CHECK (key ~ '^[A-Z]{2,5}$');

COMMIT;
