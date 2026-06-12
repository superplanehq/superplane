BEGIN;

-- Organizations created before the github+google default still have only ["github"].
UPDATE organizations
SET allowed_providers = '["github", "google"]'::jsonb
WHERE allowed_providers = '["github"]'::jsonb
  AND deleted_at IS NULL;

COMMIT;
