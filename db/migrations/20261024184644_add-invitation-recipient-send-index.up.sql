CREATE INDEX organization_invitation_sends_email_sent_at_idx
    ON organization_invitation_sends (email, sent_at);
