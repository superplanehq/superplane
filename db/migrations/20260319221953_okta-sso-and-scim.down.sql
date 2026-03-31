BEGIN;

DROP TABLE IF EXISTS organization_scim_user_mappings;
DROP TABLE IF EXISTS organization_okta_idp;
ALTER TABLE accounts DROP COLUMN IF EXISTS managed_account;

COMMIT;
