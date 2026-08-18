--
-- Inline concurrency configuration for a node. Each field defaults
-- independently, so all-NULL means the default behavior: the node runs
-- one execution at a time in its implicit queue (named after the node
-- ID).
--
ALTER TABLE workflow_nodes ADD COLUMN concurrency_key text;
ALTER TABLE workflow_nodes ADD COLUMN concurrency_max integer;

ALTER TABLE workflow_nodes
    ADD CONSTRAINT workflow_nodes_concurrency_max_check
    CHECK (concurrency_max >= 1);
