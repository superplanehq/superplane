CREATE TABLE factory_human_review_publication_leases (
    pull_request_id uuid PRIMARY KEY REFERENCES factory_pull_requests(id) ON DELETE CASCADE,
    token uuid NOT NULL,
    expires_at timestamptz NOT NULL
);
