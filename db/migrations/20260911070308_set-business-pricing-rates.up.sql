BEGIN;

-- Existing installs stored the old 20% default. A later admin choice of 2000
-- cannot be distinguished from that default, so it also moves to 10%.
UPDATE installation_llm_settings
SET markup_bps = 1000, updated_at = NOW()
WHERE markup_bps = 2000;

-- Tiny per-minute prices do not divide into integer micros per second.
-- 3 micros/s bills ~10% under $0.0002/min; 2 micros/s bills ~20% over $0.0001/min.
INSERT INTO usage_price_books (version, effective_at)
VALUES ('2026-09-09.1', TIMESTAMPTZ '2026-09-09 00:00:00+00');

INSERT INTO usage_price_book_rates (
  version, usage_kind, match_key, match_mode,
  input_cents_per_million, output_cents_per_million,
  cache_read_cents_per_million, cache_write_cents_per_million, reasoning_cents_per_million,
  micros_per_second
)
SELECT
  '2026-09-09.1',
  usage_kind,
  match_key,
  match_mode,
  input_cents_per_million,
  output_cents_per_million,
  cache_read_cents_per_million,
  cache_write_cents_per_million,
  reasoning_cents_per_million,
  micros_per_second
FROM usage_price_book_rates
WHERE version = (
  SELECT version FROM usage_price_books
  WHERE version <> '2026-09-09.1'
  ORDER BY effective_at DESC
  LIMIT 1
)
AND usage_kind = 'model';

INSERT INTO usage_price_book_rates (
  version, usage_kind, match_key, match_mode,
  input_cents_per_million, output_cents_per_million,
  cache_read_cents_per_million, cache_write_cents_per_million, reasoning_cents_per_million
) VALUES
  ('2026-09-09.1', 'model', 'gemini-2.5-pro', 'prefix', 125, 1000, 13, 0, 0),
  ('2026-09-09.1', 'model', 'gemini-2.5-flash', 'prefix', 15, 60, 2, 0, 0),
  ('2026-09-09.1', 'model', 'gemini-2.0-flash', 'prefix', 10, 40, 1, 0, 0),
  ('2026-09-09.1', 'model', 'gemini-1.5-pro', 'prefix', 125, 500, 13, 0, 0),
  ('2026-09-09.1', 'model', 'gemini-1.5-flash', 'prefix', 8, 30, 1, 0, 0),
  ('2026-09-09.1', 'model', 'gemini-flash', 'prefix', 15, 60, 2, 0, 0),
  ('2026-09-09.1', 'model', 'gemini-pro', 'prefix', 125, 1000, 13, 0, 0),
  ('2026-09-09.1', 'model', 'gemini', 'prefix', 15, 60, 2, 0, 0),
  ('2026-09-09.1', 'model', 'grok-3-mini', 'prefix', 30, 50, 3, 0, 0),
  ('2026-09-09.1', 'model', 'grok-3', 'prefix', 300, 1500, 30, 0, 0),
  ('2026-09-09.1', 'model', 'grok-2', 'prefix', 200, 1000, 20, 0, 0),
  ('2026-09-09.1', 'model', 'grok', 'prefix', 300, 1500, 30, 0, 0),
  ('2026-09-09.1', 'model', 'deepseek-reasoner', 'prefix', 55, 219, 6, 0, 0),
  ('2026-09-09.1', 'model', 'deepseek-r1', 'prefix', 55, 219, 6, 0, 0),
  ('2026-09-09.1', 'model', 'deepseek-chat', 'prefix', 27, 110, 3, 0, 0),
  ('2026-09-09.1', 'model', 'deepseek-v3', 'prefix', 27, 110, 3, 0, 0),
  ('2026-09-09.1', 'model', 'deepseek', 'prefix', 27, 110, 3, 0, 0),
  ('2026-09-09.1', 'model', 'qwen-max', 'prefix', 160, 640, 16, 0, 0),
  ('2026-09-09.1', 'model', 'qwen-plus', 'prefix', 40, 120, 4, 0, 0),
  ('2026-09-09.1', 'model', 'qwen-turbo', 'prefix', 5, 20, 1, 0, 0),
  ('2026-09-09.1', 'model', 'qwen3', 'prefix', 30, 90, 3, 0, 0),
  ('2026-09-09.1', 'model', 'qwen', 'prefix', 40, 120, 4, 0, 0),
  ('2026-09-09.1', 'model', 'kimi-k2', 'prefix', 60, 250, 6, 0, 0),
  ('2026-09-09.1', 'model', 'moonshot', 'prefix', 120, 120, 12, 0, 0),
  ('2026-09-09.1', 'model', 'kimi', 'prefix', 60, 250, 6, 0, 0)
ON CONFLICT (version, usage_kind, match_key, match_mode) DO NOTHING;

INSERT INTO usage_price_book_rates (
  version, usage_kind, match_key, match_mode, micros_per_second
) VALUES
  ('2026-09-09.1', 'compute', 'e1-tiny-amd64', 'exact', 3),
  ('2026-09-09.1', 'compute', 'e1-tiny-arm64', 'exact', 2),
  ('2026-09-09.1', 'compute', 'e1-large-amd64', 'exact', 70),
  ('2026-09-09.1', 'compute', 'e1-large-arm64', 'exact', 50),
  ('2026-09-09.1', 'compute', 'local', 'exact', 0);

COMMIT;
