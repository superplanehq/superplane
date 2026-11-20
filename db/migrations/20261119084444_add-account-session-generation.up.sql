ALTER TABLE accounts ADD COLUMN session_generation BIGINT NOT NULL DEFAULT 0
    CHECK (session_generation >= 0);
