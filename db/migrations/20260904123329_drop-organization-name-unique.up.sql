BEGIN;

-- Only the slug (organizations_slug_active_key) needs to stay unique among
-- active organizations. Names may repeat: for example, onboarding creates an
-- organization named after a GitHub owner and keeps that name stable across
-- slug-collision retries, so two organizations can legitimately share a name.
ALTER TABLE organizations DROP CONSTRAINT IF EXISTS organizations_name_key;

COMMIT;
