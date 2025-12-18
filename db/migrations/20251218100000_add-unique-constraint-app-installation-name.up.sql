begin;

-- Add unique constraint to ensure app installation names are unique per organization
CREATE UNIQUE INDEX idx_app_installations_org_name_unique
  ON app_installations(organization_id, installation_name);

commit;
