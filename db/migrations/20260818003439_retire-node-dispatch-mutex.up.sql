--
-- The node state column no longer acts as a dispatch mutex. Scheduling
-- is derived from execution counts per resolved queue, so any node left
-- in 'processing' goes back to 'ready'. The 'error' state keeps its
-- meaning.
--
UPDATE workflow_nodes SET state = 'ready' WHERE state = 'processing';

--
-- The resolved queue name for a queue item. NULL means the name was not
-- resolved yet; the queue worker resolves it on first touch and persists
-- it here so expressions are evaluated exactly once per item.
--
ALTER TABLE workflow_node_queue_items ADD COLUMN queue_name character varying(256);

--
-- The resolved queue name the execution occupies a slot in. Queues are
-- private to a node, so capacity is derived by counting non-terminal
-- executions per (workflow_id, node_id, queue_name), backed by this
-- partial index.
--
ALTER TABLE workflow_node_executions ADD COLUMN queue_name character varying(256);

CREATE INDEX idx_workflow_node_executions_active_queue
    ON workflow_node_executions (workflow_id, node_id, queue_name)
    WHERE state IN ('pending', 'started', 'cancelling');

--
-- Executions that are in flight when this migration runs must occupy a
-- slot in their node's queue, or capacity checks would not see them and
-- nodes could exceed their concurrency max until those executions
-- finish. No node has a concurrency key yet, so every node's queue is
-- its implicit node-ID queue. Terminal executions never enter capacity
-- counts and stay untouched.
--
UPDATE workflow_node_executions
    SET queue_name = node_id
    WHERE state IN ('pending', 'started', 'cancelling');
