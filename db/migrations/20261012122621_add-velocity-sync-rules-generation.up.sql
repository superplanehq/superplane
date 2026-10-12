-- A stored generation below the worker constant means classification rules
-- changed. Bitbucket syncs then rebuild the full history once, so app merges
-- already saved as people output leave the report.
ALTER TABLE public.factory_velocity_syncs
	ADD COLUMN rules_generation integer NOT NULL DEFAULT 0;
