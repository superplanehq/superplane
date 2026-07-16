ALTER TABLE users
  ADD COLUMN api_key_expires_at timestamp,
  ADD COLUMN api_key_canvas_ids jsonb NOT NULL DEFAULT '[]'::jsonb;
