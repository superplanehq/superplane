BEGIN;

-- Onboarding stored the product label "GitHub" as the integration name.
-- Secret lookup matches app_installations.installation_name, so replace that
-- label with the GitHub install bound to the workspace. A real install named
-- "GitHub" is left unchanged. Prompt text that contains the word GitHub stays.

CREATE FUNCTION rewrite_github_integration_label(value jsonb, installation_name text)
RETURNS jsonb
LANGUAGE plpgsql
AS $$
DECLARE
  object_key text;
  element jsonb;
  rewritten jsonb;
BEGIN
  IF value IS NULL OR jsonb_typeof(value) = 'null' THEN
    RETURN value;
  END IF;

  IF jsonb_typeof(value) = 'array' THEN
    rewritten := '[]'::jsonb;
    FOR element IN SELECT item FROM jsonb_array_elements(value) AS item
    LOOP
      rewritten := rewritten || jsonb_build_array(
        rewrite_github_integration_label(element, installation_name)
      );
    END LOOP;
    RETURN rewritten;
  END IF;

  IF jsonb_typeof(value) <> 'object' THEN
    RETURN value;
  END IF;

  IF value->>'source' = 'integration' AND value->'integration'->>'name' = 'GitHub' THEN
    value := jsonb_set(value, '{integration,name}', to_jsonb(installation_name), false);
  END IF;

  rewritten := '{}'::jsonb;
  FOR object_key IN SELECT jsonb_object_keys(value)
  LOOP
    rewritten := rewritten || jsonb_build_object(
      object_key,
      rewrite_github_integration_label(value->object_key, installation_name)
    );
  END LOOP;
  RETURN rewritten;
END;
$$;

CREATE TEMP TABLE github_label_targets ON COMMIT DROP AS
SELECT
  workflow.id AS workflow_id,
  install.id::text AS integration_id,
  install.installation_name
FROM workflows AS workflow
JOIN factories AS factory ON factory.id = workflow.factory_id
JOIN app_installations AS install
  ON install.organization_id = workflow.organization_id
 AND install.deleted_at IS NULL
 AND install.app_name = 'github'
 AND install.id::text = btrim(factory.onboarding_config->>'vcs_integration_id')
WHERE install.installation_name <> 'GitHub';

UPDATE workflow_nodes AS node
SET
  configuration = rewritten.configuration,
  updated_at = NOW()
FROM (
  SELECT
    current_node.workflow_id,
    current_node.node_id,
    rewrite_github_integration_label(current_node.configuration, target.installation_name) AS configuration
  FROM workflow_nodes AS current_node
  JOIN github_label_targets AS target ON target.workflow_id = current_node.workflow_id
  WHERE current_node.configuration::text LIKE '%"GitHub"%'
) AS rewritten
WHERE node.workflow_id = rewritten.workflow_id
  AND node.node_id = rewritten.node_id
  AND node.configuration IS DISTINCT FROM rewritten.configuration;

UPDATE workflow_versions AS version
SET
  nodes = rewritten.nodes,
  updated_at = NOW()
FROM (
  SELECT
    current_version.id,
    COALESCE((
      SELECT jsonb_agg(
        CASE
          WHEN jsonb_typeof(item.node->'configuration') = 'object' THEN
            jsonb_set(
              CASE
                WHEN item.node->'integration'->>'name' = 'GitHub'
                 AND COALESCE(item.node->'integration'->>'id', '') IN ('', target.integration_id)
                THEN jsonb_set(item.node, '{integration,name}', to_jsonb(target.installation_name), false)
                ELSE item.node
              END,
              '{configuration}',
              rewrite_github_integration_label(item.node->'configuration', target.installation_name),
              false
            )
          WHEN item.node->'integration'->>'name' = 'GitHub'
           AND COALESCE(item.node->'integration'->>'id', '') IN ('', target.integration_id)
          THEN jsonb_set(item.node, '{integration,name}', to_jsonb(target.installation_name), false)
          ELSE item.node
        END
        ORDER BY item.ordinality
      )
      FROM jsonb_array_elements(current_version.nodes) WITH ORDINALITY AS item(node, ordinality)
    ), '[]'::jsonb) AS nodes
  FROM workflow_versions AS current_version
  JOIN github_label_targets AS target ON target.workflow_id = current_version.workflow_id
  WHERE current_version.nodes::text LIKE '%"GitHub"%'
) AS rewritten
WHERE version.id = rewritten.id
  AND version.nodes IS DISTINCT FROM rewritten.nodes;

DROP FUNCTION rewrite_github_integration_label(jsonb, text);

COMMIT;
