ALTER TABLE runner_fleets ADD COLUMN provisioning_cooldown_until timestamptz;

CREATE INDEX runner_tasks_organization_active_idx
    ON runner_tasks (organization_id)
    WHERE backend = 'integrated' AND state IN ('reserved', 'running');

CREATE INDEX runner_tasks_fleet_queued_creation_idx
    ON runner_tasks (fleet_id, created_at, id)
    WHERE backend = 'integrated' AND state = 'queued' AND runner_id IS NULL;
