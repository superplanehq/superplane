BEGIN;

ALTER TABLE organizations
  ADD COLUMN created_by_account_id UUID REFERENCES accounts(id);

UPDATE organizations o
SET created_by_account_id = first_user.account_id
FROM (
  SELECT DISTINCT ON (organization_id) organization_id, account_id
    FROM users
  WHERE type = 'human'
    AND account_id IS NOT NULL
  ORDER BY organization_id, created_at ASC, id ASC
) AS first_user
WHERE o.id = first_user.organization_id
  AND o.created_by_account_id IS NULL;

CREATE INDEX index_organizations_on_created_by_account_id
  ON organizations (created_by_account_id);

COMMIT;
