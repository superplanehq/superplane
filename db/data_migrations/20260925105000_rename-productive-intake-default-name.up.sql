BEGIN;

-- Rename Productive task intakes that still use the old default canvas name.
-- A customized name stays as the user saved it. If the new default is already
-- taken in that workspace, use the next free "Productive tasks (n)" name.
WITH renamed AS (
  SELECT
    workflow.id,
    CASE
      WHEN NOT EXISTS (
        SELECT 1
        FROM workflows AS existing
        WHERE existing.factory_id = workflow.factory_id
          AND existing.deleted_at IS NULL
          AND existing.id <> workflow.id
          AND existing.name = 'Productive tasks'
      ) THEN 'Productive tasks'
      ELSE (
        SELECT format('Productive tasks (%s)', suffix)
        FROM generate_series(2, 100) AS suffix
        WHERE NOT EXISTS (
          SELECT 1
          FROM workflows AS existing
          WHERE existing.factory_id = workflow.factory_id
            AND existing.deleted_at IS NULL
            AND existing.id <> workflow.id
            AND existing.name = format('Productive tasks (%s)', suffix)
        )
        ORDER BY suffix
        LIMIT 1
      )
    END AS new_name
  FROM workflows AS workflow
  JOIN factory_intakes AS intake ON intake.canvas_id = workflow.id
  WHERE intake.source = 'productive-tasks'
    AND workflow.deleted_at IS NULL
    AND workflow.name = 'Productive.io tasks'
)
UPDATE workflows AS workflow
SET
  name = renamed.new_name,
  description = CASE
    WHEN workflow.description = 'Create a work order when a Productive.io task is created.'
      THEN 'Create a work order when a Productive task is created.'
    ELSE workflow.description
  END,
  updated_at = NOW()
FROM renamed
WHERE workflow.id = renamed.id
  AND renamed.new_name IS NOT NULL;

-- A user may have renamed the intake and left the default description.
UPDATE workflows AS workflow
SET
  description = 'Create a work order when a Productive task is created.',
  updated_at = NOW()
FROM factory_intakes AS intake
WHERE intake.canvas_id = workflow.id
  AND intake.source = 'productive-tasks'
  AND workflow.deleted_at IS NULL
  AND workflow.description = 'Create a work order when a Productive.io task is created.';

COMMIT;
