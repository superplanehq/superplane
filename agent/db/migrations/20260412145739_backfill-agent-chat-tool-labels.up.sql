BEGIN;

CREATE OR REPLACE FUNCTION agent_hardcoded_tool_display_label(p_tool text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
  SELECT CASE lower(trim(p_tool))
    WHEN 'get_canvas' THEN 'Reading canvas'
    WHEN 'get_canvas_memory' THEN 'Loading canvas notes'
    WHEN 'get_canvas_shape' THEN 'Reading canvas structure'
    WHEN 'get_canvas_details' THEN 'Reading canvas details'
    WHEN 'get_node_details' THEN 'Reading node details'
    WHEN 'list_node_events' THEN 'Listing node events'
    WHEN 'list_node_executions' THEN 'Listing node executions'
    WHEN 'list_available_blocks' THEN 'Listing available components'
    WHEN 'list_components' THEN 'List components'
    WHEN 'list_triggers' THEN 'List triggers'
    WHEN 'list_decision_patterns' THEN 'List decision patterns'
    WHEN 'search_decision_patterns' THEN 'Search decision patterns'
    WHEN 'get_decision_pattern' THEN 'Get decision pattern'
    WHEN 'describe_component' THEN 'Describe component'
    WHEN 'describe_trigger' THEN 'Describe trigger'
    WHEN 'list_org_integrations' THEN 'List org integrations'
    WHEN 'list_available_integrations' THEN 'List available integrations'
    WHEN 'list_integration_resources' THEN 'List integration resources'
    ELSE initcap(
      regexp_replace(
        regexp_replace(trim(lower(COALESCE(p_tool, ''))), '_', ' ', 'g'),
        '-',
        ' ',
        'g'
      )
    )
  END;
$$;

UPDATE agent_chat_messages AS m
SET
  message = jsonb_set(m.message, '{parts}', sub.new_parts, true),
  updated_at = NOW()
FROM (
  SELECT
    m2.id,
    COALESCE(
      (
        SELECT jsonb_agg(rebuilt.elem ORDER BY rebuilt.ord)
        FROM (
          SELECT
            t.ordinality AS ord,
            CASE
              WHEN t.elem->>'part_kind' = 'tool-return'
                AND COALESCE(t.elem->'metadata', '{}'::jsonb)->>'superplane_display_label' IS NULL
                AND lower(trim(COALESCE(t.elem->>'tool_name', ''))) NOT IN (
                  'final_result',
                  'return_canvasanswer',
                  'canvasanswer'
                )
              THEN jsonb_set(
                t.elem,
                '{metadata}',
                COALESCE(t.elem->'metadata', '{}'::jsonb)
                  || jsonb_build_object(
                    'superplane_display_label',
                    agent_hardcoded_tool_display_label(t.elem->>'tool_name')
                  ),
                true
              )
              ELSE t.elem
            END AS elem
          FROM jsonb_array_elements(COALESCE(m2.message->'parts', '[]'::jsonb))
            WITH ORDINALITY AS t (elem, ordinality)
        ) AS rebuilt
      ),
      m2.message->'parts'
    ) AS new_parts
  FROM agent_chat_messages AS m2
  WHERE jsonb_typeof(m2.message) = 'object'
    AND jsonb_typeof(m2.message->'parts') = 'array'
) AS sub
WHERE m.id = sub.id
  AND sub.new_parts IS DISTINCT FROM m.message->'parts';

DROP FUNCTION agent_hardcoded_tool_display_label(text);

COMMIT;
