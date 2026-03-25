BEGIN;

-- For existing installations that already went through owner setup,
-- promote the earliest created account to installation admin.
-- This is a one-time data migration; new installations get this
-- automatically via the setup_owner flow.

UPDATE accounts
SET installation_admin = TRUE
WHERE id = (
    SELECT id
    FROM accounts
    ORDER BY created_at ASC
    LIMIT 1
)
AND NOT installation_admin;

COMMIT;
