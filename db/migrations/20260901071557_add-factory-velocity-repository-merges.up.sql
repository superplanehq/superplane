BEGIN;

-- Merged pull requests of a workspace repository that this SuperPlane instance
-- did not open. The sync excludes numbers already in factory_pull_requests.
--
-- source is who wrote the change: 'people' is a person, 'agent' is a merge
-- whose commit carries the SuperPlane agent co-author trailer. People counts
-- come from people rows. SuperPlane counts come from factory_pull_requests plus
-- agent rows (agent work opened from another instance).
--
-- merged_at is the merge instant, so charts bucket by the viewer's day.
CREATE TABLE factory_velocity_repository_merges (
  id UUID PRIMARY KEY,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
  factory_id UUID NOT NULL REFERENCES factories(id) ON DELETE CASCADE,
  repository TEXT NOT NULL,
  number BIGINT NOT NULL,
  source TEXT NOT NULL DEFAULT 'people',
  author_login TEXT NOT NULL,
  author_name TEXT NOT NULL DEFAULT '',
  author_avatar_url TEXT NOT NULL DEFAULT '',
  merged_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT factory_velocity_repository_merges_number_positive CHECK (number > 0),
  CONSTRAINT factory_velocity_repository_merges_source_valid CHECK (source IN ('people', 'agent'))
);

CREATE UNIQUE INDEX idx_factory_velocity_repository_merges_factory_repo_number
  ON factory_velocity_repository_merges (factory_id, repository, number);

CREATE INDEX idx_factory_velocity_repository_merges_factory_merged_at
  ON factory_velocity_repository_merges (factory_id, merged_at DESC);

-- One row per workspace. repository is the repo the stored merges belong to, so
-- a change of app repository restarts the backfill. synced_at and updated_at
-- also act as the claim for a running sync.
CREATE TABLE factory_velocity_syncs (
  factory_id UUID PRIMARY KEY REFERENCES factories(id) ON DELETE CASCADE,
  repository TEXT NOT NULL DEFAULT '',
  synced_at TIMESTAMPTZ,
  backfilled_from TIMESTAMPTZ,
  error TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_factory_velocity_syncs_synced_at
  ON factory_velocity_syncs (synced_at NULLS FIRST);

COMMIT;
