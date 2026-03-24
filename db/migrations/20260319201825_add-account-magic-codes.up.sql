BEGIN;

CREATE TABLE account_magic_codes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email VARCHAR(255) NOT NULL,
  code_hash VARCHAR(64) NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  used_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
  verify_attempts INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_account_magic_codes_email ON account_magic_codes(email);
CREATE INDEX idx_account_magic_codes_email_code_hash ON account_magic_codes(email, code_hash);

COMMIT;
