begin;

CREATE TABLE app_installation_requests (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  app_installation_id uuid NOT NULL,
  state CHARACTER VARYING(32) NOT NULL,
  type CHARACTER VARYING(32) NOT NULL,
  run_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (app_installation_id) REFERENCES app_installations(id) ON DELETE CASCADE
);

CREATE INDEX idx_app_installation_requests_state_run_at
ON app_installation_requests(state, run_at) WHERE state = 'pending';

CREATE INDEX idx_app_installation_requests_installation_id
ON app_installation_requests(app_installation_id);

commit;
