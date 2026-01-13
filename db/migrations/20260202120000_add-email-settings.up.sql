CREATE TABLE public.email_settings (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    provider character varying(50) NOT NULL,
    smtp_host character varying(255),
    smtp_port integer,
    smtp_username character varying(255),
    smtp_password bytea,
    smtp_from_name character varying(255),
    smtp_from_email character varying(255),
    smtp_use_tls boolean DEFAULT false NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT email_settings_pkey PRIMARY KEY (id),
    CONSTRAINT email_settings_provider_key UNIQUE (provider)
);
