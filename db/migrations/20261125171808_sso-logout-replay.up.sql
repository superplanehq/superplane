CREATE TABLE sso_logout_replays (
    connection_id uuid NOT NULL REFERENCES sso_connections(id) ON DELETE CASCADE,
    jti           text NOT NULL,
    expires_at    timestamptz NOT NULL,
    PRIMARY KEY (connection_id, jti)
);

CREATE INDEX sso_logout_replays_expires_at ON sso_logout_replays (expires_at);
