BEGIN;

CREATE TABLE agent_canvas_markdown_memory (
    canvas_id uuid PRIMARY KEY,
    markdown_body text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT NOW()
);

COMMIT;
