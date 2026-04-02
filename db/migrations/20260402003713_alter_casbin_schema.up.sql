BEGIN;

ALTER TABLE casbin_rule
  ALTER COLUMN ptype DROP NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_casbin_rule_unique
    ON public.casbin_rule USING btree (ptype, v0, v1, v2, v3, v4, v5);

COMMIT;