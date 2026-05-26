BEGIN;

ALTER TABLE organizations
    ADD COLUMN enabled_experimental_features JSONB NOT NULL DEFAULT '[]'::jsonb;

COMMIT;
