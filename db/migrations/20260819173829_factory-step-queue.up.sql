--
-- Line-step admission control: a work order that becomes ready for a
-- step at its maxParallelism waits in the step's queue. Queued work is
-- stored as a queue item, not an execution; the run and execution are
-- only created when the step admits the work order, and the queue item
-- is deleted.
--
-- A queue item belongs to a line dispatch (the traversal that is
-- waiting). A dispatch waits at no more than one step at a time, so
-- line_dispatch_id is unique.
--
CREATE TABLE factory_work_order_queue_items (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id  UUID NOT NULL,
    factory_id       UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
    work_order_id    UUID NOT NULL REFERENCES factory_work_orders(id) ON DELETE RESTRICT,
    line_id          UUID NOT NULL REFERENCES factory_lines(id) ON DELETE RESTRICT,
    line_dispatch_id UUID NOT NULL UNIQUE REFERENCES factory_work_order_line_dispatches(id) ON DELETE RESTRICT,
    step_index       INTEGER NOT NULL,
    step_name        TEXT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

--
-- Admission pops the oldest item per (line, step); the work-order and
-- factory indexes serve API reads and cleanup.
--
CREATE INDEX idx_factory_work_order_queue_items_step
    ON factory_work_order_queue_items (line_id, step_index, created_at);

CREATE INDEX idx_factory_work_order_queue_items_order
    ON factory_work_order_queue_items (work_order_id);

CREATE INDEX idx_factory_work_order_queue_items_factory
    ON factory_work_order_queue_items (factory_id);
