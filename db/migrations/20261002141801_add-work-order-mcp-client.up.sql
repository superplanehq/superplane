BEGIN;

ALTER TABLE factory_work_orders
  ADD COLUMN mcp_client_id TEXT,
  ADD COLUMN mcp_client_name TEXT;

COMMIT;
