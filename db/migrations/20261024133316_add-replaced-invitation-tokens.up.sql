CREATE TABLE organization_invitation_replaced_tokens (
    token_hash varchar(64) PRIMARY KEY,
    invitation_id uuid NOT NULL REFERENCES organization_invitations(id) ON DELETE CASCADE
);

CREATE INDEX organization_invitation_replaced_tokens_invitation_id_idx
    ON organization_invitation_replaced_tokens (invitation_id);
