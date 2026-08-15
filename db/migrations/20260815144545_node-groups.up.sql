--
-- Node groups: a drawn group of nodes on the canvas that acts as a
-- queue. Group definitions ({ id, nodes, maxParallelism }) are versioned
-- with the spec; group membership is materialized onto workflow_nodes so
-- the queue worker can gate dispatch without loading the canvas version.
--
ALTER TABLE workflow_versions ADD COLUMN node_groups jsonb;
ALTER TABLE workflow_nodes ADD COLUMN group_id character varying(128);

--
-- Holders of group-queue slots. A row means the run currently holds a
-- slot in the group's queue. Node-queue capacity is derived from
-- execution counts and has no rows here.
--
CREATE TABLE workflow_queue_slots (
    workflow_id uuid NOT NULL,
    group_id    character varying(128) NOT NULL,
    run_id      uuid NOT NULL,
    acquired_at timestamp NOT NULL,
    PRIMARY KEY (workflow_id, group_id, run_id)
);

CREATE INDEX idx_workflow_queue_slots_run ON workflow_queue_slots (workflow_id, run_id);
