begin;

CREATE TABLE stage_event_approvals (
  id             uuid NOT NULL DEFAULT uuid_generate_v4(),
  stage_event_id uuid NOT NULL,
  approved_at    TIMESTAMP NOT NULL,
  approved_by    uuid NOT NULL,

  PRIMARY KEY (id),
  UNIQUE (stage_event_id, approved_by),
  FOREIGN KEY (stage_event_id) REFERENCES stage_events(id)
);

CREATE INDEX uix_stage_event_approvals_events ON stage_event_approvals USING btree (stage_event_id);

commit;
