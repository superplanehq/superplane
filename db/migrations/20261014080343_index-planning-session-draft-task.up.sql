CREATE INDEX CONCURRENTLY idx_factory_planning_sessions_draft_work_order
ON public.factory_planning_sessions (draft_work_order_id)
INCLUDE (canvas_run_id)
WHERE draft_work_order_id IS NOT NULL;
