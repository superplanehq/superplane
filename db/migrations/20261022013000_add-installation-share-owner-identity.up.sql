ALTER TABLE public.installation_metadata
    ADD COLUMN IF NOT EXISTS share_owner_identity boolean DEFAULT false NOT NULL;
