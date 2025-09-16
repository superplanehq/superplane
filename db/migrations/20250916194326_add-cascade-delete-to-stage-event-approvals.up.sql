begin;

-- Drop the existing foreign key constraint
ALTER TABLE stage_event_approvals
DROP CONSTRAINT stage_event_approvals_stage_event_id_fkey;

-- Add the foreign key constraint with CASCADE deletion
ALTER TABLE stage_event_approvals
ADD CONSTRAINT stage_event_approvals_stage_event_id_fkey
FOREIGN KEY (stage_event_id) REFERENCES stage_events(id) ON DELETE CASCADE;

commit;