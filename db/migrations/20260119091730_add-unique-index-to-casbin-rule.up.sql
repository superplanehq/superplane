BEGIN;

-- Migration: Add unique index to casbin_rule table
-- This index is automatically created by gorm-adapter at runtime.
-- Adding it explicitly to the migration ensures schema consistency
-- and prevents duplicate RBAC policy entries.

CREATE UNIQUE INDEX IF NOT EXISTS idx_casbin_rule ON casbin_rule(ptype, v0, v1, v2, v3, v4, v5);

COMMIT;
