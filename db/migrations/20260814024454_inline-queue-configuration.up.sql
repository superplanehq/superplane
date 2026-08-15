--
-- Queue configuration moves inline to nodes and groups:
-- - workflow_nodes.queue becomes a jsonb QueueSpec
--   ({ key, maxParallelism, autoCancel }); the old text value was the
--   queue name template, which maps to the spec's key. The old reserved
--   name 'none' maps to maxParallelism 0 (unlimited).
-- - workflow_versions.queue_rules is dropped; canvas-level rules no
--   longer exist.
-- - workflow_queue_slots is keyed by group instead of queue name; group
--   queues no longer have names.
--
ALTER TABLE workflow_nodes
    ALTER COLUMN queue TYPE jsonb
    USING CASE
        WHEN queue IS NULL THEN NULL
        WHEN queue = 'none' THEN '{"maxParallelism": 0}'::jsonb
        ELSE jsonb_build_object('key', queue)
    END;

UPDATE workflow_versions
SET nodes = COALESCE((
    SELECT jsonb_agg(
        CASE
            WHEN NOT (n ? 'queue') THEN n
            WHEN n->>'queue' = 'none' THEN jsonb_set(n, '{queue}', '{"maxParallelism": 0}'::jsonb)
            ELSE jsonb_set(n, '{queue}', jsonb_build_object('key', n->>'queue'))
        END)
    FROM jsonb_array_elements(nodes) n
), '[]'::jsonb)
WHERE jsonb_typeof(nodes) = 'array';

UPDATE workflow_versions
SET node_groups = COALESCE((
    SELECT jsonb_agg(g - 'queue') FROM jsonb_array_elements(node_groups) g
), '[]'::jsonb)
WHERE jsonb_typeof(node_groups) = 'array';

ALTER TABLE workflow_versions DROP COLUMN queue_rules;

DELETE FROM workflow_queue_slots;
ALTER TABLE workflow_queue_slots DROP CONSTRAINT workflow_queue_slots_pkey;
ALTER TABLE workflow_queue_slots DROP COLUMN queue_name;
ALTER TABLE workflow_queue_slots ADD PRIMARY KEY (workflow_id, group_id, run_id);

--
-- Queues are private to a node, so capacity counts are scoped by node.
--
DROP INDEX idx_workflow_node_executions_active_queue;
CREATE INDEX idx_workflow_node_executions_active_queue
    ON workflow_node_executions (workflow_id, node_id, queue_name)
    WHERE state IN ('pending', 'started', 'cancelling');
