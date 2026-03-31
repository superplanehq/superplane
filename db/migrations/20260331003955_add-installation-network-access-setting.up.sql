ALTER TABLE public.installation_metadata
    ADD COLUMN allow_private_network_access boolean DEFAULT false NOT NULL;
