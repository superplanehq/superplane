BEGIN;

CREATE TABLE IF NOT EXISTS account_survey_responses (
  id              UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
  account_id      UUID         NOT NULL UNIQUE REFERENCES accounts(id) ON DELETE CASCADE,
  survey_type     VARCHAR(64)  NOT NULL DEFAULT 'signup',
  skipped         BOOLEAN      NOT NULL DEFAULT FALSE,
  source_channel  VARCHAR(64)  NULL,
  source_other    TEXT         NULL,
  role            VARCHAR(64)  NULL,
  use_case        TEXT         NULL,
  created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_account_survey_responses_source_channel
  ON account_survey_responses(source_channel)
  WHERE skipped = FALSE;

COMMIT;
