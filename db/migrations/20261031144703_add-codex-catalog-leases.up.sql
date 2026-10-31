BEGIN;

ALTER TABLE factory_codex_leases
    DROP CONSTRAINT factory_codex_leases_purpose_check,
    DROP CONSTRAINT factory_codex_leases_scope_check,
    ADD CONSTRAINT factory_codex_leases_purpose_check
        CHECK (purpose IN ('login', 'run', 'catalog')),
    ADD CONSTRAINT factory_codex_leases_scope_check
        CHECK ((purpose IN ('login', 'catalog') AND factory_id IS NULL)
            OR (purpose = 'run' AND factory_id IS NOT NULL));

-- Keep locally revoked catalog leases unique until task cleanup removes them.
CREATE UNIQUE INDEX factory_codex_leases_catalog_idx
    ON factory_codex_leases (connection_id)
    WHERE purpose = 'catalog';

COMMIT;
