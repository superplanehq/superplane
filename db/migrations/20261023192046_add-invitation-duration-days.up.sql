ALTER TABLE installation_metadata
    ADD COLUMN invitation_duration_days integer NOT NULL DEFAULT 7
    CHECK (invitation_duration_days BETWEEN 1 AND 365);
