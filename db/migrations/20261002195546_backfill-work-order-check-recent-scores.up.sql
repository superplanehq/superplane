UPDATE factory_work_order_checks
SET recent_scores = '[]'::jsonb
WHERE recent_scores IS NULL;

ALTER TABLE factory_work_order_checks
  ALTER COLUMN recent_scores SET DEFAULT '[]'::jsonb;
