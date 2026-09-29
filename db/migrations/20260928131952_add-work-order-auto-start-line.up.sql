BEGIN;

ALTER TABLE factory_work_orders
  ADD COLUMN auto_start_line_id UUID REFERENCES factory_lines(id) ON DELETE SET NULL;

COMMIT;
