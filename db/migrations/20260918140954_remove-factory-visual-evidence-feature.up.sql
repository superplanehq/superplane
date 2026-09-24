BEGIN;

UPDATE organizations
SET enabled_experimental_features = enabled_experimental_features - 'factory_visual_evidence'
WHERE enabled_experimental_features ? 'factory_visual_evidence';

COMMIT;
