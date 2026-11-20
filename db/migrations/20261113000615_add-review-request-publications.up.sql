CREATE TABLE review_request_publications (
    integration_id uuid NOT NULL REFERENCES app_installations(id) ON DELETE CASCADE,
    key text NOT NULL,
    record jsonb NOT NULL,
    created_at timestamp NOT NULL DEFAULT NOW(),
    updated_at timestamp NOT NULL DEFAULT NOW(),
    PRIMARY KEY (integration_id, key)
);

ALTER TABLE app_messages ADD COLUMN source_run_id uuid;
ALTER TABLE workflow_events ADD COLUMN source_run_id uuid;
