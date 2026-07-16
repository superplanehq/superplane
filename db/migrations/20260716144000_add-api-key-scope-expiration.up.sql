ALTER TABLE users
  ADD COLUMN service_account_expires_at timestamp,
  ADD COLUMN service_account_canvas_ids jsonb NOT NULL DEFAULT '[]'::jsonb;
