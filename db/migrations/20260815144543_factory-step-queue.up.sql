--
-- Line-step admission control: a work order that becomes ready for a step
-- at its maxParallelism waits in the step's queue. Queued work is stored
-- as a queue item, not an execution; an execution (and its run) is only
-- created when the step admits the work order.
--
CREATE TABLE factory_work_order_queue_items (
    id              uuid DEFAULT uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    factory_id      uuid NOT NULL,
    work_order_id   uuid NOT NULL,
    line_id         uuid NOT NULL,
    step_index      integer NOT NULL,
    step_name       text NOT NULL,
    created_at      timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id)
);

--
-- Admission pops the oldest item per (line, step); the work-order index
-- serves API reads and active-work checks.
--
CREATE INDEX idx_factory_work_order_queue_items_step ON factory_work_order_queue_items (line_id, step_index, created_at);
CREATE INDEX idx_factory_work_order_queue_items_order ON factory_work_order_queue_items (work_order_id);
