--
-- Canvas-level queue configuration, versioned with the spec:
-- queue_rules is an ordered list of { match, maxParallelism, autoCancel },
-- node_groups is a list of { id, nodes, queue } group definitions.
--
ALTER TABLE workflow_versions ADD COLUMN queue_rules jsonb;
ALTER TABLE workflow_versions ADD COLUMN node_groups jsonb;
