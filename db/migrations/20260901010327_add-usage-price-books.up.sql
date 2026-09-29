BEGIN;

CREATE TABLE usage_price_books (
  version      TEXT PRIMARY KEY,
  effective_at TIMESTAMPTZ NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE usage_price_book_rates (
  id                            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  version                       TEXT NOT NULL REFERENCES usage_price_books(version),
  usage_kind                    TEXT NOT NULL CHECK (usage_kind IN ('model', 'compute')),
  match_key                     TEXT NOT NULL,
  match_mode                    TEXT NOT NULL CHECK (match_mode IN ('exact', 'prefix', 'family')),
  input_cents_per_million       BIGINT NOT NULL DEFAULT 0,
  output_cents_per_million      BIGINT NOT NULL DEFAULT 0,
  cache_read_cents_per_million  BIGINT NOT NULL DEFAULT 0,
  cache_write_cents_per_million BIGINT NOT NULL DEFAULT 0,
  reasoning_cents_per_million   BIGINT NOT NULL DEFAULT 0,
  micros_per_second             BIGINT NOT NULL DEFAULT 0,
  UNIQUE (version, usage_kind, match_key, match_mode)
);

INSERT INTO usage_price_books (version, effective_at)
VALUES ('2026-08-31.1', TIMESTAMPTZ '2026-08-31 00:00:00+00');

INSERT INTO usage_price_book_rates (
  version, usage_kind, match_key, match_mode,
  input_cents_per_million, output_cents_per_million,
  cache_read_cents_per_million, cache_write_cents_per_million, reasoning_cents_per_million
) VALUES
  ('2026-08-31.1', 'model', 'claude-opus', 'prefix', 1500, 7500, 150, 1875, 0),
  ('2026-08-31.1', 'model', 'claude-sonnet', 'prefix', 300, 1500, 30, 375, 0),
  ('2026-08-31.1', 'model', 'claude-haiku', 'prefix', 80, 400, 8, 100, 0),
  ('2026-08-31.1', 'model', 'gpt-4o-mini', 'prefix', 15, 60, 1, 0, 0),
  ('2026-08-31.1', 'model', 'gpt-4o', 'prefix', 250, 1000, 25, 0, 0),
  ('2026-08-31.1', 'model', 'gpt-5-mini', 'prefix', 25, 200, 2, 0, 0),
  ('2026-08-31.1', 'model', 'gpt-5', 'prefix', 125, 1000, 12, 0, 0),
  ('2026-08-31.1', 'model', 'o3-mini', 'prefix', 110, 440, 11, 0, 0),
  ('2026-08-31.1', 'model', 'o3', 'prefix', 2000, 8000, 200, 0, 0),
  ('2026-08-31.1', 'model', 'o4-mini', 'prefix', 110, 440, 11, 0, 0),
  ('2026-08-31.1', 'model', 'opus', 'family', 1500, 7500, 150, 1875, 0),
  ('2026-08-31.1', 'model', 'sonnet', 'family', 300, 1500, 30, 375, 0),
  ('2026-08-31.1', 'model', 'haiku', 'family', 80, 400, 8, 100, 0);

INSERT INTO usage_price_book_rates (
  version, usage_kind, match_key, match_mode, micros_per_second
) VALUES
  ('2026-08-31.1', 'compute', 'e1-large-amd64', 'exact', 0),
  ('2026-08-31.1', 'compute', 'e1-large-arm64', 'exact', 0),
  ('2026-08-31.1', 'compute', 'e1-tiny-amd64', 'exact', 0),
  ('2026-08-31.1', 'compute', 'e1-tiny-arm64', 'exact', 0),
  ('2026-08-31.1', 'compute', 'local', 'exact', 0);

COMMIT;
