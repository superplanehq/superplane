BEGIN;

-- The table stored only merges people wrote by hand, so anything a machine
-- merged was dropped at sync time. That lost SuperPlane's own output: the agent
-- opens pull requests through a GitHub App, and factory_pull_requests only knows
-- the pull requests this instance opened itself. Merges an agent made through
-- another instance counted in neither series.
--
-- The table now holds every merge of the repository that SuperPlane did not
-- open, and source says who wrote it, so one row set feeds both series.
ALTER TABLE factory_velocity_people_merges
  RENAME TO factory_velocity_repository_merges;

ALTER INDEX idx_factory_velocity_people_merges_factory_repo_number
  RENAME TO idx_factory_velocity_repository_merges_factory_repo_number;

ALTER INDEX idx_factory_velocity_people_merges_factory_merged_at
  RENAME TO idx_factory_velocity_repository_merges_factory_merged_at;

ALTER TABLE factory_velocity_repository_merges
  RENAME CONSTRAINT factory_velocity_people_merges_number_positive
  TO factory_velocity_repository_merges_number_positive;

-- 'people' is a person's own work. 'agent' is a merge whose commit carries the
-- SuperPlane agent co-author trailer, which identifies agent output whichever
-- instance opened the pull request.
ALTER TABLE factory_velocity_repository_merges
  ADD COLUMN source TEXT NOT NULL DEFAULT 'people';

ALTER TABLE factory_velocity_repository_merges
  ADD CONSTRAINT factory_velocity_repository_merges_source_valid
  CHECK (source IN ('people', 'agent'));

-- Every stored row predates the source column and was classified as a person's
-- work, so the next sync must recompute them. Clearing the sync state makes the
-- worker backfill the full window again instead of only recomputing recent days.
UPDATE factory_velocity_syncs
  SET synced_at = NULL, backfilled_from = NULL;

COMMIT;
