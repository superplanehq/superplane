BEGIN;

ALTER TABLE organization_llm_credit_grants
  ADD COLUMN expires_at timestamptz;

UPDATE organization_llm_credit_grants
SET expires_at = NOW() + INTERVAL '14 days'
WHERE kind = 'welcome'
  AND expires_at IS NULL;

COMMIT;
