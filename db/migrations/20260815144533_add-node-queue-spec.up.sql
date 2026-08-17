--
-- Inline concurrency configuration for a node, as a ConcurrencySpec
-- object ({ key, max, autoCancel }). NULL means the node uses its
-- implicit queue: named after the node ID, max 1.
--
ALTER TABLE workflow_nodes ADD COLUMN concurrency jsonb;
