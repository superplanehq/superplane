ALTER TABLE factory_work_order_checks
  ADD COLUMN source character varying(16) DEFAULT 'automation' NOT NULL;
