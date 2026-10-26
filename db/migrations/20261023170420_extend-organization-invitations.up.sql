ALTER TABLE organization_invitations
    ADD COLUMN role varchar(255) NOT NULL DEFAULT 'org_operator',
    ADD COLUMN token_hash varchar(64) UNIQUE,
    ADD COLUMN expires_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ADD COLUMN accepted_at timestamp without time zone,
    ADD COLUMN accepted_by uuid REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN sent_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP;

UPDATE organization_invitations SET state = 'expired' WHERE state = 'pending';
UPDATE organization_invitations SET sent_at = COALESCE(updated_at, created_at, CURRENT_TIMESTAMP);

CREATE UNIQUE INDEX organization_invitations_pending_email_idx
    ON organization_invitations (organization_id, lower(email)) WHERE state = 'pending';
