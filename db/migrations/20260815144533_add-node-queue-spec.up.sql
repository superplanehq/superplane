--
-- Inline queue configuration for a node, as a QueueSpec object
-- ({ key, maxParallelism, autoCancel }). NULL means the node uses its
-- implicit queue: named after the node ID, maxParallelism 1.
--
ALTER TABLE workflow_nodes ADD COLUMN queue jsonb;
