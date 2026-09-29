BEGIN;

-- Apps that belong to a workspace only have to be unique inside that
-- workspace. Every workspace installs the same templates ("Plan", "Implement",
-- "PR Closure"), so an organization-wide constraint made the second workspace
-- fall back to "Plan (2)", "Plan (3)" and so on.
--
-- Apps that do not belong to a workspace stay unique per organization, because
-- the organization is the only scope they have.
--
-- Both indexes only cover live rows. A soft-deleted app keeps its name for
-- history and no longer blocks a new app from taking that name.

ALTER TABLE workflows DROP CONSTRAINT workflows_organization_id_name_key;

CREATE UNIQUE INDEX workflows_organization_id_name_active_key
  ON workflows (organization_id, name)
  WHERE factory_id IS NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX workflows_factory_id_name_active_key
  ON workflows (factory_id, name)
  WHERE factory_id IS NOT NULL AND deleted_at IS NULL;

COMMIT;
