BEGIN;

CREATE TABLE canvas_data (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    canvas_id uuid NOT NULL,
    key character varying(256) NOT NULL,
    value text NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT canvas_data_pkey PRIMARY KEY (id),
    CONSTRAINT canvas_data_canvas_id_fkey FOREIGN KEY (canvas_id) REFERENCES workflows(id) ON DELETE CASCADE
);

CREATE INDEX canvas_data_canvas_id_key_created_at_idx ON canvas_data (canvas_id, key, created_at DESC);

COMMIT;
