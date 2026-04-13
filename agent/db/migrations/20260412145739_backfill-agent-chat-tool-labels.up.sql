BEGIN;

UPDATE agent_chat_messages AS m
SET
  message = jsonb_set(
    m.message,
    '{parts}',
    COALESCE(
      (
        SELECT jsonb_agg(
          CASE
            WHEN e->>'part_kind' = 'tool-return' THEN
              jsonb_set(
                e,
                '{metadata}',
                COALESCE(e->'metadata', '{}'::jsonb)
                  || jsonb_build_object('superplane_display_label', COALESCE(e->>'tool_name', '')),
                true
              )
            ELSE e
          END
          ORDER BY n
        )
        FROM jsonb_array_elements(m.message->'parts') WITH ORDINALITY AS t (e, n)
      ),
      m.message->'parts'
    ),
    true
  ),
  updated_at = NOW()
WHERE jsonb_typeof(m.message->'parts') = 'array';

COMMIT;
