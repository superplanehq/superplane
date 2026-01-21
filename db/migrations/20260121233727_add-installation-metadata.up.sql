CREATE TABLE public.installation_metadata (
    id integer NOT NULL,
    installation_id character varying(64) NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT installation_metadata_pkey PRIMARY KEY (id),
    CONSTRAINT installation_metadata_singleton CHECK (id = 1)
);
