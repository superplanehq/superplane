CREATE TABLE factory_merge_confidence_publication_claims (
    pull_request_id uuid PRIMARY KEY REFERENCES factory_pull_requests(id) ON DELETE CASCADE,
    owner_id uuid NOT NULL,
    lease_until timestamp with time zone NOT NULL
);
