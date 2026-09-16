ALTER TABLE public.user_notification_settings
    ADD COLUMN browser_workspace_scope character varying(50) DEFAULT 'none' NOT NULL,
    ADD COLUMN browser_workspace_filters jsonb DEFAULT '[]'::jsonb NOT NULL,
    ADD COLUMN browser_event_types jsonb DEFAULT '[]'::jsonb NOT NULL,
    ADD COLUMN browser_show_while_viewing boolean DEFAULT true NOT NULL;
