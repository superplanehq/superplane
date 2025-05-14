begin;

CREATE TABLE stage_executions (
  id             uuid NOT NULL DEFAULT uuid_generate_v4(),
  stage_id       uuid NOT NULL,
  stage_event_id uuid NOT NULL,
  reference_id   CHARACTER VARYING(64) NOT NULL,
  state          CHARACTER VARYING(64) NOT NULL,
  result         CHARACTER VARYING(64) NOT NULL,
  tags           jsonb,
  created_at     TIMESTAMP NOT NULL,
  updated_at     TIMESTAMP NOT NULL,
  started_at     TIMESTAMP,
  finished_at    TIMESTAMP,

  PRIMARY KEY (id),
  FOREIGN KEY (stage_id) REFERENCES stages(id),
  FOREIGN KEY (stage_event_id) REFERENCES stage_events(id)
);

CREATE INDEX uix_stage_executions_stage ON stage_executions USING btree (stage_id);
CREATE INDEX uix_stage_executions_events ON stage_executions USING btree (stage_event_id);

commit;
