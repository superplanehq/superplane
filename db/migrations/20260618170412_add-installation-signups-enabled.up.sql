ALTER TABLE public.installation_metadata
    ADD COLUMN signups_enabled boolean DEFAULT true NOT NULL;
