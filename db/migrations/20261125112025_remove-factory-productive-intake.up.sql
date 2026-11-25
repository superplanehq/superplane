BEGIN;

UPDATE organizations
SET enabled_experimental_features = enabled_experimental_features - 'factory_productive_intake'
WHERE enabled_experimental_features ? 'factory_productive_intake';

COMMIT;
