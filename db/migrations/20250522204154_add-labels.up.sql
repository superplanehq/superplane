begin;

ALTER TABLE stage_executions RENAME COLUMN tags TO labels;
ALTER TABLE stage_connections ADD COLUMN labels jsonb NOT NULL DEFAULT '{}';
ALTER TABLE stage_events ADD COLUMN labels jsonb NOT NULL DEFAULT '{}';

CREATE TABLE stage_execution_labels (
  id           uuid NOT NULL DEFAULT uuid_generate_v4(),
  source_id    uuid NOT NULL,
  source_type  CHARACTER VARYING(64) NOT NULL,
  execution_id uuid NOT NULL,
  name         CHARACTER VARYING(64) NOT NULL,
  value        CHARACTER VARYING(64) NOT NULL,
  created_at   TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (execution_id) REFERENCES stage_executions(id)
);

commit;