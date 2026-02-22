begin;

CREATE TABLE scripts (
    id              uuid NOT NULL DEFAULT uuid_generate_v4(),
    organization_id uuid NOT NULL,
    name            CHARACTER VARYING(128) NOT NULL,
    label           CHARACTER VARYING(256) NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    source          TEXT NOT NULL DEFAULT '',
    manifest        JSONB NOT NULL DEFAULT '{}'::jsonb,
    status          CHARACTER VARYING(32) NOT NULL DEFAULT 'draft',
    created_by      uuid REFERENCES users(id),
    created_at      TIMESTAMP NOT NULL,
    updated_at      TIMESTAMP NOT NULL,

    PRIMARY KEY (id),
    UNIQUE (organization_id, name)
);

CREATE INDEX idx_scripts_organization_id ON scripts(organization_id);

commit;
