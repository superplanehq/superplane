BEGIN;

-- Add username column to users table
ALTER TABLE users
  ADD COLUMN username VARCHAR(255) UNIQUE;

-- Update users with username from GitHub account provider
UPDATE users
  SET username = ap.username
  FROM account_providers ap
  WHERE users.id = ap.user_id
    AND ap.provider = 'github'
    AND ap.username IS NOT NULL;

-- Make username NOT NULL after populating existing records
ALTER TABLE users
  ALTER COLUMN username SET NOT NULL;

-- Create index on username
CREATE INDEX idx_users_username ON users(username);

COMMIT;