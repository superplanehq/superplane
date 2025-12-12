begin;

ALTER TABLE webhooks
  ADD COLUMN app_installation_id uuid,
  ADD FOREIGN KEY (app_installation_id) REFERENCES app_installations(id);

CREATE INDEX idx_webhooks_app_installation_id ON webhooks(app_installation_id);

commit;