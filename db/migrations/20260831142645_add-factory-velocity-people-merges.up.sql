BEGIN;

-- Merged pull requests of a workspace's connected repository that SuperPlane
-- did not open. The background sync excludes anything matching
-- factory_pull_requests, so the two tables hold disjoint sets and the velocity
-- report needs no join to tell people output from SuperPlane output.
--
-- merged_at is the exact merge instant rather than a calendar day, so the chart
-- can bucket merges by the viewer's day without a midnight rounding error.
CREATE TABLE factory_velocity_people_merges (
  id UUID PRIMARY KEY,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
  factory_id UUID NOT NULL REFERENCES factories(id) ON DELETE CASCADE,
  repository TEXT NOT NULL,
  number BIGINT NOT NULL,
  author_login TEXT NOT NULL,
  author_name TEXT NOT NULL DEFAULT '',
  author_avatar_url TEXT NOT NULL DEFAULT '',
  merged_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT factory_velocity_people_merges_number_positive CHECK (number > 0)
);

CREATE UNIQUE INDEX idx_factory_velocity_people_merges_factory_repo_number
  ON factory_velocity_people_merges (factory_id, repository, number);

CREATE INDEX idx_factory_velocity_people_merges_factory_merged_at
  ON factory_velocity_people_merges (factory_id, merged_at DESC);

-- One row per workspace tracking how far its repository sync has reached.
-- repository is the repository the stored merges were collected for, so a
-- workspace that changes its app repository restarts the backfill.
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
