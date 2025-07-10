BEGIN;

-- Add username column with default value generated from name for existing users
ALTER TABLE users
  ADD COLUMN username VARCHAR(255) UNIQUE 
  DEFAULT (LOWER(REPLACE(name, ' ', '_')));

CREATE INDEX idx_users_username ON users(username);

COMMIT;