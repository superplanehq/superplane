ALTER TABLE factory_work_orders
    ADD COLUMN creator_run_id uuid REFERENCES workflow_runs(id) ON DELETE SET NULL;
