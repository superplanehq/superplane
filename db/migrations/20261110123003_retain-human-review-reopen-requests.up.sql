ALTER TABLE factory_pull_requests ADD COLUMN human_review_request_id uuid;

CREATE INDEX idx_factory_pull_requests_human_review_pending
  ON factory_pull_requests (id)
  WHERE human_review_request_id IS NOT NULL;
