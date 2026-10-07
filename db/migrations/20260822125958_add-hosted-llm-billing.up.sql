BEGIN;

CREATE TABLE IF NOT EXISTS installation_llm_settings (
  id                      INTEGER PRIMARY KEY DEFAULT 1,
  welcome_grant_cents     BIGINT NOT NULL DEFAULT 5000,
  markup_bps              INTEGER NOT NULL DEFAULT 2000,
  warning_threshold_bps   INTEGER NOT NULL DEFAULT 2000,
  created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT installation_llm_settings_singleton CHECK (id = 1),
  CONSTRAINT installation_llm_settings_welcome_non_negative CHECK (welcome_grant_cents >= 0),
  CONSTRAINT installation_llm_settings_markup_non_negative CHECK (markup_bps >= 0),
  CONSTRAINT installation_llm_settings_warning_range CHECK (warning_threshold_bps >= 0 AND warning_threshold_bps <= 10000)
);

INSERT INTO installation_llm_settings (id, welcome_grant_cents, markup_bps, warning_threshold_bps)
VALUES (1, 5000, 2000, 2000)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS hosted_llm_providers (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  provider        TEXT NOT NULL,
  enabled         BOOLEAN NOT NULL DEFAULT FALSE,
  api_key         BYTEA,
  base_url        TEXT NOT NULL DEFAULT '',
  allowed_models  JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (provider),
  CONSTRAINT hosted_llm_providers_known CHECK (provider IN ('anthropic', 'openai', 'openrouter'))
);

CREATE TABLE IF NOT EXISTS organization_llm_credit_grants (
  id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id   UUID NOT NULL,
  kind              TEXT NOT NULL,
  amount_micros     BIGINT NOT NULL,
  note              TEXT NOT NULL DEFAULT '',
  actor_account_id  UUID,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT organization_llm_credit_grants_kind CHECK (kind IN ('welcome', 'admin')),
  CONSTRAINT organization_llm_credit_grants_amount_positive CHECK (amount_micros > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_org_llm_credit_grants_welcome
  ON organization_llm_credit_grants (organization_id)
  WHERE kind = 'welcome';

CREATE INDEX IF NOT EXISTS idx_org_llm_credit_grants_org
  ON organization_llm_credit_grants (organization_id, created_at DESC);

CREATE TABLE IF NOT EXISTS organization_llm_settings (
  organization_id  UUID PRIMARY KEY,
  markup_bps       INTEGER,
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT organization_llm_settings_markup_non_negative CHECK (markup_bps IS NULL OR markup_bps >= 0)
);

ALTER TABLE llm_usage_events
  ADD COLUMN IF NOT EXISTS provider_cost_micros BIGINT NOT NULL DEFAULT 0;

UPDATE llm_usage_events
SET provider_cost_micros = cost_micros
WHERE provider_cost_micros = 0 AND cost_micros <> 0;

COMMIT;
