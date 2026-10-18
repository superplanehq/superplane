ALTER TABLE account_providers
  DROP COLUMN access_token,
  DROP COLUMN refresh_token,
  DROP COLUMN token_expires_at;
