--
-- Holders of group-queue slots. A row means the run currently holds a
-- slot in the group's queue. Node-queue capacity is derived from
-- execution counts and has no rows here.
--
CREATE TABLE workflow_queue_slots (
    workflow_id uuid NOT NULL,
    queue_name  character varying(256) NOT NULL,
    group_id    character varying(128) NOT NULL,
    run_id      uuid NOT NULL,
    acquired_at timestamp NOT NULL,
    PRIMARY KEY (workflow_id, queue_name, run_id)
);

CREATE INDEX idx_workflow_queue_slots_run ON workflow_queue_slots (workflow_id, run_id);
