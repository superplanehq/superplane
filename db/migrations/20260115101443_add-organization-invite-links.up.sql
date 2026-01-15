CREATE TABLE public.organization_invite_links (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    token uuid NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE ONLY public.organization_invite_links
    ADD CONSTRAINT organization_invite_links_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.organization_invite_links
    ADD CONSTRAINT organization_invite_links_organization_id_key UNIQUE (organization_id);

ALTER TABLE ONLY public.organization_invite_links
    ADD CONSTRAINT organization_invite_links_token_key UNIQUE (token);

ALTER TABLE ONLY public.organization_invite_links
    ADD CONSTRAINT organization_invite_links_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;

INSERT INTO public.organization_invite_links (organization_id, token, enabled, created_at, updated_at)
SELECT id, gen_random_uuid(), true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP FROM public.organizations;
