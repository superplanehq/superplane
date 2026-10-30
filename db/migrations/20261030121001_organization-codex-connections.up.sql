-- Drain leases before this migration. Preserve encrypted connection IDs.
BEGIN;
LOCK TABLE factory_codex_connections, factory_codex_leases IN ACCESS EXCLUSIVE MODE;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM factory_codex_leases) THEN
        RAISE EXCEPTION 'Drain all Codex leases before migrating personal connections';
    END IF;
END $$;

CREATE TEMP TABLE codex_connection_survivors ON COMMIT DROP AS
SELECT organization_id, owner_user_id,
       COALESCE(
           (array_agg(id ORDER BY id) FILTER (
               WHERE status = 'ready' AND credential_version > 0
                 AND octet_length(bundle_ciphertext) > 0 AND revocation_pending_at IS NULL
           ))[1],
           (array_agg(id ORDER BY id))[1]
       ) AS id,
       count(*) FILTER (
           WHERE status = 'ready' AND credential_version > 0
             AND octet_length(bundle_ciphertext) > 0 AND revocation_pending_at IS NULL
       ) AS usable
FROM factory_codex_connections
GROUP BY organization_id, owner_user_id;

-- Multiple usable bundles may belong to different accounts. Require sign-in.
-- Clear obsolete credentials and usage before removing duplicate rows.
UPDATE factory_codex_connections c SET
    status = CASE WHEN s.usable > 1 THEN 'reconnect_required' ELSE 'disconnected' END,
    resume_status = '', verification_url = '', verification_code = '', account_label = '',
    bundle_ciphertext = NULL, credential_version = 0, final_checkpoint_version = 0,
    usage_limits = '[]', available_models = '[]', usage_observed_at = NULL,
    reservation_generation = reservation_generation + 1
FROM codex_connection_survivors s
WHERE c.organization_id = s.organization_id AND c.owner_user_id = s.owner_user_id
  AND (c.id <> s.id OR s.usable <> 1);

DELETE FROM factory_codex_connections c USING codex_connection_survivors s
WHERE c.organization_id = s.organization_id AND c.owner_user_id = s.owner_user_id AND c.id <> s.id;

ALTER TABLE factory_codex_connections
    DROP CONSTRAINT factory_codex_connections_factory_owner_key,
    DROP COLUMN factory_id,
    ADD CONSTRAINT factory_codex_connections_organization_owner_key UNIQUE (organization_id, owner_user_id);

ALTER TABLE factory_codex_leases
    ADD COLUMN factory_id uuid REFERENCES factories(id) ON DELETE RESTRICT,
    ADD CONSTRAINT factory_codex_leases_scope_check
        CHECK ((purpose = 'login' AND factory_id IS NULL) OR (purpose = 'run' AND factory_id IS NOT NULL));
COMMIT;
