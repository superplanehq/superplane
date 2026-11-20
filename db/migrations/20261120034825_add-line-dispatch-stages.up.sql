ALTER TABLE factory_work_order_line_dispatches
ADD COLUMN stages jsonb NOT NULL DEFAULT '[]'::jsonb;
