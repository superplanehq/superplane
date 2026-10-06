BEGIN;

-- Companies that could see Jira, Productive, or Datadog keep that access
-- on the organization Features tab.
UPDATE organizations AS organizations
SET enabled_experimental_features = organizations.enabled_experimental_features || flags.feature_ids
FROM (
  SELECT
    intake_catalog_organizations.organization_id,
    jsonb_agg(DISTINCT to_jsonb(mapping.feature_id)) AS feature_ids
  FROM intake_catalog_organizations
  JOIN organizations AS existing
    ON existing.id = intake_catalog_organizations.organization_id
  JOIN (VALUES
    ('jira-issues', 'factory_jira_intake'),
    ('productive-tasks', 'factory_productive_intake'),
    ('datadog', 'factory_datadog_intake')
  ) AS mapping(entry_key, feature_id)
    ON intake_catalog_organizations.entry_key = mapping.entry_key
  WHERE NOT (existing.enabled_experimental_features ? mapping.feature_id)
  GROUP BY intake_catalog_organizations.organization_id
) AS flags
WHERE organizations.id = flags.organization_id;

-- Entries that were opened to all companies keep that access by enabling the
-- corresponding features for every organization.
UPDATE organizations AS organizations
SET enabled_experimental_features = organizations.enabled_experimental_features || flags.feature_ids
FROM (
  SELECT
    organizations.id AS organization_id,
    jsonb_agg(DISTINCT to_jsonb(mapping.feature_id)) AS feature_ids
  FROM organizations
  JOIN intake_catalog_entries
    ON intake_catalog_entries.enabled_for_all IS TRUE
  JOIN (VALUES
    ('jira-issues', 'factory_jira_intake'),
    ('productive-tasks', 'factory_productive_intake'),
    ('datadog', 'factory_datadog_intake')
  ) AS mapping(entry_key, feature_id)
    ON intake_catalog_entries.key = mapping.entry_key
  WHERE NOT (organizations.enabled_experimental_features ? mapping.feature_id)
  GROUP BY organizations.id
) AS flags
WHERE organizations.id = flags.organization_id;

DROP TABLE intake_catalog_organizations;

ALTER TABLE intake_catalog_entries DROP COLUMN enabled_for_all;

COMMIT;
