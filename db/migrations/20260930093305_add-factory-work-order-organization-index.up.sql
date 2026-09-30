CREATE INDEX idx_factory_work_orders_organization_state_result
ON public.factory_work_orders USING btree (organization_id, state, result);
