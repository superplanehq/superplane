ALTER TABLE factory_codex_connections
    DROP CONSTRAINT factory_codex_connections_factory_key,
    ADD CONSTRAINT factory_codex_connections_factory_owner_key UNIQUE (factory_id, owner_user_id),
    ADD COLUMN revocation_pending_at timestamptz;

ALTER TABLE factory_codex_leases ADD COLUMN revoked_at timestamptz;
CREATE INDEX factory_codex_connections_revocation_pending_idx
    ON factory_codex_connections (revocation_pending_at)
    WHERE revocation_pending_at IS NOT NULL;

ALTER TABLE factory_work_orders ADD COLUMN pending_handoff jsonb;
