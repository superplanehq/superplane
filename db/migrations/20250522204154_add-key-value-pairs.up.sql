begin;

ALTER TABLE stage_executions RENAME COLUMN tags TO kv;
ALTER TABLE stage_connections ADD COLUMN kv_defs jsonb NOT NULL DEFAULT '{}';
ALTER TABLE stage_events ADD COLUMN kv jsonb NOT NULL DEFAULT '{}';

CREATE TABLE stage_kvs (
  id         uuid NOT NULL DEFAULT uuid_generate_v4(),
  stage_id   uuid NOT NULL,
  key        CHARACTER VARYING(64) NOT NULL,
  value      CHARACTER VARYING(64) NOT NULL,
  created_at TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (stage_id) REFERENCES stages(id)
);

commit;