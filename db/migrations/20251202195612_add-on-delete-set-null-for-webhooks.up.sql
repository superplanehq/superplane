BEGIN;

ALTER TABLE workflow_nodes DROP CONSTRAINT workflow_nodes_webhook_id_fkey;

ALTER TABLE workflow_nodes
  ADD CONSTRAINT workflow_nodes_webhook_id_fkey
  FOREIGN KEY (webhook_id) REFERENCES webhooks(id) ON DELETE SET NULL;

COMMIT;