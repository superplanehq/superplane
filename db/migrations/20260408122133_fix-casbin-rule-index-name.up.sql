BEGIN;

DROP INDEX IF EXISTS public.idx_casbin_rule_unique;

CREATE UNIQUE INDEX IF NOT EXISTS idx_casbin_rule
    ON public.casbin_rule USING btree (ptype, v0, v1, v2, v3, v4, v5);

COMMIT;
