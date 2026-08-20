ALTER TABLE factory_work_orders
  ADD COLUMN IF NOT EXISTS status_note jsonb;
