CREATE TABLE IF NOT EXISTS sso_identity_sessions (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id   uuid NOT NULL REFERENCES sso_connections(id) ON DELETE CASCADE,
    identity_id     uuid NOT NULL REFERENCES sso_identities(id) ON DELETE CASCADE,
    idp_session_id  text NOT NULL,
    expires_at      timestamptz NOT NULL,
    revoked_at      timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS sso_identity_sessions_active_session
    ON sso_identity_sessions (connection_id, idp_session_id)
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS sso_identity_sessions_identity
    ON sso_identity_sessions (identity_id)
    WHERE revoked_at IS NULL;
