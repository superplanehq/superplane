--
-- Node groups: a drawn group of nodes on the canvas that acts as a
-- queue. Group definitions ({ id, nodes, max }) are versioned with the
-- spec and materialized on publish, so the queue worker can gate
-- dispatch without loading the canvas version: one workflow_node_groups
-- row per group, plus membership on workflow_nodes.group_id.
--
ALTER TABLE workflow_versions ADD COLUMN node_groups jsonb;
ALTER TABLE workflow_nodes ADD COLUMN group_id character varying(128);

CREATE TABLE workflow_node_groups (
    workflow_id uuid NOT NULL,
    group_id    character varying(128) NOT NULL,
    max         integer,
    PRIMARY KEY (workflow_id, group_id),
    CONSTRAINT workflow_node_groups_max_check CHECK (max >= 1)
);

--
-- Holders of group-queue slots. A row means the run currently holds a
-- slot in the group's queue. Node-queue capacity is derived from
-- execution counts and has no rows here. Removing a group on publish
-- drops its held slots.
--
CREATE TABLE workflow_queue_slots (
    workflow_id uuid NOT NULL,
    group_id    character varying(128) NOT NULL,
    run_id      uuid NOT NULL,
    acquired_at timestamp NOT NULL,
    PRIMARY KEY (workflow_id, group_id, run_id),
    FOREIGN KEY (workflow_id, group_id)
        REFERENCES workflow_node_groups (workflow_id, group_id)
        ON DELETE CASCADE
);

CREATE INDEX idx_workflow_queue_slots_run ON workflow_queue_slots (workflow_id, run_id);
