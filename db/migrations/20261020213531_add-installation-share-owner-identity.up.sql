ALTER TABLE public.installation_metadata
    ADD COLUMN share_owner_identity boolean DEFAULT false NOT NULL;
