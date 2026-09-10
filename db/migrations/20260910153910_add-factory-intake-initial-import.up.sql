ALTER TABLE factory_intakes
  ADD COLUMN initial_import_status VARCHAR(32) NOT NULL DEFAULT 'unspecified',
  ADD COLUMN initial_import_item_count INTEGER,
  ADD CONSTRAINT factory_intakes_initial_import_status_valid
    CHECK (initial_import_status IN ('unspecified', 'pending', 'completed', 'failed', 'skipped')),
  ADD CONSTRAINT factory_intakes_initial_import_count_valid
    CHECK (
      (initial_import_status = 'completed' AND initial_import_item_count IS NOT NULL AND initial_import_item_count >= 0)
      OR (initial_import_status <> 'completed' AND initial_import_item_count IS NULL)
    );
