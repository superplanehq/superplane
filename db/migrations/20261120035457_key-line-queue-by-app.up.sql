ALTER TABLE factory_work_order_queue_items
ADD COLUMN app_id uuid;

UPDATE factory_work_order_queue_items AS queue
SET app_id = (dispatch.steps -> queue.step_index ->> 'app_id')::uuid
FROM factory_work_order_line_dispatches AS dispatch
WHERE dispatch.id = queue.line_dispatch_id;

ALTER TABLE factory_work_order_queue_items
ALTER COLUMN app_id SET NOT NULL;

DROP INDEX idx_factory_work_order_queue_items_step;

CREATE INDEX idx_factory_work_order_queue_items_app
ON factory_work_order_queue_items (line_id, app_id, created_at);
