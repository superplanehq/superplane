ALTER TABLE public.user_notification_settings
    ADD COLUMN event_types jsonb DEFAULT '[]'::jsonb NOT NULL;
