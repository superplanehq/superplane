--
-- The resolved queue name for the item. NULL means the name was not
-- resolved yet; the queue worker resolves it on first touch and persists
-- it here so expressions are evaluated exactly once per item.
--
ALTER TABLE workflow_node_queue_items ADD COLUMN queue_name character varying(256);
