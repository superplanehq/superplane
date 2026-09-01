BEGIN;

-- Catalog VM rates: tiny ≈ $0.50/hour (139 micros/s), large ≈ $2.00/hour (556 micros/s).
-- local stays 0. Markup, wallet debit, and runner hard-stop stay out of this change.

INSERT INTO usage_price_books (version, effective_at)
VALUES ('2026-08-31.2', TIMESTAMPTZ '2026-08-31 12:00:00+00');

INSERT INTO usage_price_book_rates (
  version, usage_kind, match_key, match_mode,
  input_cents_per_million, output_cents_per_million,
  cache_read_cents_per_million, cache_write_cents_per_million, reasoning_cents_per_million,
  micros_per_second
)
SELECT
  '2026-08-31.2',
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
WHERE version = '2026-08-31.1' AND usage_kind = 'model';

INSERT INTO usage_price_book_rates (
  version, usage_kind, match_key, match_mode, micros_per_second
) VALUES
  ('2026-08-31.2', 'compute', 'e1-large-amd64', 'exact', 556),
  ('2026-08-31.2', 'compute', 'e1-large-arm64', 'exact', 556),
  ('2026-08-31.2', 'compute', 'e1-tiny-amd64', 'exact', 139),
  ('2026-08-31.2', 'compute', 'e1-tiny-arm64', 'exact', 139),
  ('2026-08-31.2', 'compute', 'local', 'exact', 0);

COMMIT;
