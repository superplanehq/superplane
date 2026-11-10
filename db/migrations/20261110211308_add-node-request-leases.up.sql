ALTER TABLE workflow_node_requests
    ADD COLUMN lease_id uuid,
    ADD COLUMN lease_expires_at timestamp with time zone;
