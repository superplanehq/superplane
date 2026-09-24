BEGIN;

ALTER TABLE usage_price_books
  ADD COLUMN is_current BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE usage_price_books
SET is_current = TRUE
WHERE version = (
  SELECT version
  FROM usage_price_books
  ORDER BY effective_at DESC
  LIMIT 1
);

CREATE UNIQUE INDEX usage_price_books_one_current
  ON usage_price_books ((TRUE))
  WHERE is_current;

COMMIT;
