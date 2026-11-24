UPDATE organizations
SET enabled_experimental_features =
    (enabled_experimental_features - 'factory_custom_stages') ||
    CASE WHEN enabled_experimental_features ? 'factory_custom_automations'
        THEN '[]'::jsonb
        ELSE '["factory_custom_automations"]'::jsonb
    END
WHERE enabled_experimental_features ? 'factory_custom_stages';
