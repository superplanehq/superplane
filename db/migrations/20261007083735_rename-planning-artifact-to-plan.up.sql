BEGIN;

UPDATE factory_work_order_artifacts
SET data = jsonb_set(
  jsonb_set(data, '{name}', '"plan.md"'::jsonb, true),
  '{title}',
  '"plan.md"'::jsonb,
  true
)
WHERE key LIKE 'spec:%'
  AND (
    data->>'name' = 'spec.md'
    OR data->>'title' = 'spec.md'
  );

COMMIT;
