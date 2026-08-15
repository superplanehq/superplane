ALTER TABLE workflow_nodes ADD COLUMN queue character varying(256);
ALTER TABLE workflow_nodes ADD COLUMN group_id character varying(128);

--
-- The node state column no longer acts as a dispatch mutex.
-- Scheduling is now derived from execution counts per queue,
-- so any node left in 'processing' goes back to 'ready'.
-- The 'error' state keeps its meaning.
--
UPDATE workflow_nodes SET state = 'ready' WHERE state = 'processing';
