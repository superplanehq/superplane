CREATE TABLE public.user_notification_settings (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    user_id uuid NOT NULL,
    enabled boolean DEFAULT false NOT NULL,
    workspace_scope character varying(50) DEFAULT 'all' NOT NULL,
    factory_ids jsonb DEFAULT '[]'::jsonb NOT NULL,
    types jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    CONSTRAINT user_notification_settings_pkey PRIMARY KEY (id),
    CONSTRAINT user_notification_settings_organization_id_user_id_key UNIQUE (organization_id, user_id)
);
