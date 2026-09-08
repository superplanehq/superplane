BEGIN;

ALTER TABLE accounts
  ADD COLUMN welcome_credit_granted_at timestamptz;

UPDATE accounts a
SET welcome_credit_granted_at = first_welcome.granted_at
FROM (
  SELECT DISTINCT ON (o.created_by_account_id)
    o.created_by_account_id AS account_id,
    g.created_at AS granted_at
  FROM organization_llm_credit_grants g
  JOIN organizations o ON o.id = g.organization_id
  WHERE g.kind = 'welcome'
    AND o.created_by_account_id IS NOT NULL
  ORDER BY o.created_by_account_id, g.created_at ASC, g.id ASC
) AS first_welcome
WHERE a.id = first_welcome.account_id
  AND a.welcome_credit_granted_at IS NULL;

COMMIT;
