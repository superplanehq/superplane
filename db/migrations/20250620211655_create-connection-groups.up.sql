BEGIN;

CREATE TABLE connection_groups (
  id         uuid NOT NULL DEFAULT uuid_generate_v4(),
  name       CHARACTER VARYING(128) NOT NULL,
  canvas_id  uuid NOT NULL,
  created_at TIMESTAMP NOT NULL,
  created_by uuid NOT NULL,
  updated_at TIMESTAMP,
  updated_by uuid,
  spec       jsonb NOT NULL DEFAULT '{}',

  PRIMARY KEY (id),
  UNIQUE (canvas_id, name),
  FOREIGN KEY (canvas_id) REFERENCES canvases(id)
);

CREATE TABLE connection_group_field_sets (
  id                  uuid NOT NULL DEFAULT uuid_generate_v4(),
  canvas_id           uuid NOT NULL,
  connection_group_id uuid NOT NULL,
  field_set           jsonb NOT NULL,
  field_set_hash      CHARACTER(64) NOT NULL,
  state               CHARACTER VARYING(64) NOT NULL,
  created_at          TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (canvas_id) REFERENCES canvases(id),
  FOREIGN KEY (connection_group_id) REFERENCES connection_groups(id)
);

CREATE TABLE connection_group_field_set_events (
  id                      uuid NOT NULL DEFAULT uuid_generate_v4(),
  connection_group_set_id uuid NOT NULL,
  event_id                uuid NOT NULL,
  source_id               uuid NOT NULL,
  source_name             CHARACTER VARYING(128) NOT NULL,
  source_type             CHARACTER VARYING(64) NOT NULL,
  received_at             TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (connection_group_set_id) REFERENCES connection_group_field_sets(id)
);

COMMIT;