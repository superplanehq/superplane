BEGIN;

ALTER TABLE workflow_node_requests
    DROP CONSTRAINT workflow_node_requests_workflow_id_node_id_fkey;

COMMIT;
