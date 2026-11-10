BEGIN;

ALTER TABLE vcs_provider_repositories
  ADD COLUMN collaborators_synced_at TIMESTAMPTZ;

-- Spread the first staleness sweep over 24 hours for repositories that
-- already have collaborators.
UPDATE vcs_provider_repositories AS repository
SET collaborators_synced_at = NOW() - random() * INTERVAL '24 hours'
WHERE EXISTS (
  SELECT 1
  FROM vcs_provider_repository_collaborators AS collaborator
  WHERE collaborator.provider = repository.provider
    AND collaborator.repository_id = repository.repository_id
);

CREATE INDEX vcs_provider_repositories_collaborators_synced_at_idx
  ON vcs_provider_repositories (provider, collaborators_synced_at);

COMMIT;
