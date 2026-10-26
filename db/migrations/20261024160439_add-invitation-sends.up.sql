CREATE TABLE organization_invitation_sends (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email varchar(255) NOT NULL,
    sent_at timestamptz NOT NULL
);

CREATE INDEX organization_invitation_sends_organization_id_sent_at_idx
    ON organization_invitation_sends (organization_id, sent_at);

CREATE INDEX organization_invitation_sends_organization_id_email_sent_at_idx
    ON organization_invitation_sends (organization_id, email, sent_at);
