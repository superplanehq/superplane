BEGIN;

-- Older databases can have newer schema migrations applied already, which can
-- cause this data migration to run before the intake catalog tables exist.
-- Create the minimal tables needed for seeding and later migration steps.
CREATE TABLE IF NOT EXISTS intake_catalog_entries (
  key varchar(64) PRIMARY KEY,
  name varchar(255) NOT NULL,
  category varchar(64) NOT NULL,
  status varchar(32) NOT NULL DEFAULT 'planned',
  status_note text NOT NULL DEFAULT '',
  enabled_for_all boolean NOT NULL DEFAULT false,
  created_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by uuid
);

CREATE TABLE IF NOT EXISTS intake_catalog_organizations (
  entry_key varchar(64) NOT NULL REFERENCES intake_catalog_entries(key) ON DELETE CASCADE ON UPDATE CASCADE,
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  created_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (entry_key, organization_id)
);

CREATE INDEX IF NOT EXISTS idx_intake_catalog_organizations_organization_id
  ON intake_catalog_organizations (organization_id);

INSERT INTO intake_catalog_entries (key, name, category, status, status_note, enabled_for_all) VALUES
  ('github-issues', 'GitHub issues', 'issue_tracking', 'ga', 'SuperPlane uses this intake every day.', false),
  ('jira-issues', 'Jira issues', 'issue_tracking', 'beta', '', false),
  ('productive-tasks', 'Productive tasks', 'issue_tracking', 'beta', '', false),
  ('linear-issues', 'Linear issues', 'issue_tracking', 'planned', '', false),
  ('notion', 'Notion', 'issue_tracking', 'planned', '', false),
  ('sentry-exceptions', 'Sentry exceptions', 'error_tracking', 'ga', 'SuperPlane uses this intake every day.', false),
  ('datadog', 'Datadog errors', 'error_tracking', 'beta', 'SuperPlane does not use Datadog. Outside testers must confirm that it works.', false),
  ('pagerduty-incidents', 'PagerDuty incidents', 'incident_management', 'alpha', '', false),
  ('dependabot-alerts', 'Dependabot alerts', 'security_alerts', 'ga', '', false),
  ('github', 'GitHub', 'repository_provider', 'ga', '', false),
  ('gitlab', 'GitLab', 'repository_provider', 'planned', '', false),
  ('bitbucket', 'Bitbucket', 'repository_provider', 'planned', '', false)
ON CONFLICT (key) DO NOTHING;

-- Keep access for organizations that had the retired per-intake features on.
INSERT INTO intake_catalog_organizations (entry_key, organization_id)
SELECT mapping.entry_key, organizations.id
FROM organizations
JOIN (VALUES
  ('factory_jira_intake', 'jira-issues'),
  ('factory_productive_intake', 'productive-tasks'),
  ('factory_datadog_intake', 'datadog')
) AS mapping(feature_id, entry_key)
  ON organizations.enabled_experimental_features ? mapping.feature_id
ON CONFLICT DO NOTHING;

UPDATE organizations
SET enabled_experimental_features = enabled_experimental_features
  - 'factory_jira_intake'
  - 'factory_productive_intake'
  - 'factory_datadog_intake'
WHERE enabled_experimental_features ?| ARRAY['factory_jira_intake', 'factory_productive_intake', 'factory_datadog_intake'];

COMMIT;
