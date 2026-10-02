BEGIN;

UPDATE organizations
SET enabled_experimental_features = enabled_experimental_features - 'factory_task_console'
WHERE enabled_experimental_features ? 'factory_task_console';

COMMIT;
