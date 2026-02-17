-- Add type column to distinguish human users from service accounts
ALTER TABLE users ADD COLUMN type varchar(50) NOT NULL DEFAULT 'human';

-- Add description column for service accounts
ALTER TABLE users ADD COLUMN description text;

-- Add created_by column to track who created a service account
ALTER TABLE users ADD COLUMN created_by uuid REFERENCES users(id);

-- Make account_id nullable (service accounts have no account)
ALTER TABLE users ALTER COLUMN account_id DROP NOT NULL;

-- Make email nullable (service accounts have no email)
ALTER TABLE users ALTER COLUMN email DROP NOT NULL;

-- Drop the old unique constraint
ALTER TABLE users DROP CONSTRAINT unique_user_in_organization;

-- Add partial unique index for human users
CREATE UNIQUE INDEX unique_human_user_in_organization
  ON users (organization_id, account_id, email)
  WHERE type = 'human';

-- Add partial unique index for service accounts (unique name per org)
CREATE UNIQUE INDEX unique_service_account_in_organization
  ON users (organization_id, name)
  WHERE type = 'service_account';
