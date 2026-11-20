ALTER TABLE factory_work_order_queue_items
ADD COLUMN stage_key text NOT NULL DEFAULT '';

UPDATE factory_work_order_queue_items AS queue
SET stage_key = COALESCE(dispatch.steps -> queue.step_index ->> 'stage_key', '')
FROM factory_work_order_line_dispatches AS dispatch
WHERE dispatch.id = queue.line_dispatch_id;

DROP INDEX idx_factory_work_order_queue_items_app;

CREATE INDEX idx_factory_work_order_queue_items_stage
ON factory_work_order_queue_items (line_id, app_id, stage_key, created_at);
