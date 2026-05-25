begin;

-- Applied only after the dedupe job has confirmed zero duplicate
-- (app_installation_id, scope_key) pairs. PostgreSQL treats NULLs in unique
-- indexes as distinct, so legacy rows with scope_key IS NULL are unaffected.
CREATE UNIQUE INDEX idx_webhooks_unique_scope
  ON webhooks(app_installation_id, scope_key)
  WHERE deleted_at IS NULL;

commit;
