BEGIN;

ALTER TABLE installation_llm_settings
  ADD COLUMN IF NOT EXISTS default_hosted_provider TEXT,
  ADD COLUMN IF NOT EXISTS default_hosted_model TEXT;

ALTER TABLE installation_llm_settings
  DROP CONSTRAINT IF EXISTS installation_llm_settings_default_model_pair;

ALTER TABLE installation_llm_settings
  ADD CONSTRAINT installation_llm_settings_default_model_pair CHECK (
    (default_hosted_provider IS NULL AND default_hosted_model IS NULL)
    OR (
      default_hosted_provider IN ('anthropic', 'openai', 'openrouter')
      AND default_hosted_model IS NOT NULL
      AND btrim(default_hosted_model) <> ''
    )
  );

UPDATE workflow_nodes
SET
  ref = jsonb_set(ref, '{component,name}', '"runnerSuperPlane"'::jsonb, true),
  configuration = (
    (configuration - 'credentials' - 'maxTurns' - 'model')
    || CASE
      WHEN COALESCE(btrim(configuration ->> 'model'), '') = '' THEN '{}'::jsonb
      ELSE jsonb_build_object(
        'model',
        'hosted::'
          || CASE ref -> 'component' ->> 'name'
            WHEN 'runnerClaudeCode' THEN 'anthropic'
            WHEN 'runnerCodex' THEN 'openai'
            WHEN 'runnerOpenRouter' THEN 'openrouter'
          END
          || '::'
          || (configuration ->> 'model')
      )
    END
  )
WHERE ref -> 'component' ->> 'name' IN ('runnerClaudeCode', 'runnerCodex', 'runnerOpenRouter')
  AND configuration -> 'credentials' ->> 'source' = 'hosted';

UPDATE workflow_versions
SET nodes = COALESCE((
  SELECT jsonb_agg(
    CASE
      WHEN elem -> 'ref' -> 'component' ->> 'name' IN ('runnerClaudeCode', 'runnerCodex', 'runnerOpenRouter')
        AND elem -> 'configuration' -> 'credentials' ->> 'source' = 'hosted'
      THEN elem
        || jsonb_build_object(
          'ref', jsonb_set(COALESCE(elem -> 'ref', '{}'::jsonb), '{component,name}', '"runnerSuperPlane"'::jsonb, true),
          'configuration',
            (COALESCE(elem -> 'configuration', '{}'::jsonb) - 'credentials' - 'maxTurns' - 'model')
            || CASE
              WHEN COALESCE(btrim(elem -> 'configuration' ->> 'model'), '') = '' THEN '{}'::jsonb
              ELSE jsonb_build_object(
                'model',
                'hosted::'
                  || CASE elem -> 'ref' -> 'component' ->> 'name'
                    WHEN 'runnerClaudeCode' THEN 'anthropic'
                    WHEN 'runnerCodex' THEN 'openai'
                    WHEN 'runnerOpenRouter' THEN 'openrouter'
                  END
                  || '::'
                  || (elem -> 'configuration' ->> 'model')
              )
            END
        )
      ELSE elem
    END
    )
  FROM jsonb_array_elements(nodes) AS elem
), '[]'::jsonb)
WHERE jsonb_typeof(nodes) = 'array';

COMMIT;
