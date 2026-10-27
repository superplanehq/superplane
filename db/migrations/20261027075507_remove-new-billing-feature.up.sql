BEGIN;

UPDATE organizations
SET enabled_experimental_features = enabled_experimental_features - 'new_billing'
WHERE enabled_experimental_features ? 'new_billing';

COMMIT;
