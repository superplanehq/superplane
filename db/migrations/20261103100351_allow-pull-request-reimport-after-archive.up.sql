ALTER TABLE factory_pull_requests ADD COLUMN IF NOT EXISTS superseded_at timestamptz;

DROP INDEX idx_factory_pull_requests_factory_provider_external;
CREATE UNIQUE INDEX idx_factory_pull_requests_factory_provider_external
ON factory_pull_requests (factory_id, provider, external_id)
WHERE external_id IS NOT NULL AND superseded_at IS NULL;

DROP INDEX idx_factory_pull_requests_factory_provider_repo_number;
CREATE UNIQUE INDEX idx_factory_pull_requests_factory_provider_repo_number
ON factory_pull_requests (factory_id, provider, repository, number)
WHERE superseded_at IS NULL;

DROP INDEX idx_factory_pull_requests_factory_url;
CREATE UNIQUE INDEX idx_factory_pull_requests_factory_url
ON factory_pull_requests (factory_id, url)
WHERE superseded_at IS NULL;
