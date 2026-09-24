BEGIN;

-- Velocity credits a GitHub login to one member per organization. The same
-- identity may link to several SuperPlane accounts when those accounts do not
-- share an organization.
DROP INDEX IF EXISTS idx_account_linked_accounts_provider_identity;

COMMIT;
