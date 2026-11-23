BEGIN;

CREATE TABLE IF NOT EXISTS factory_intake_retirements (
    canvas_id uuid PRIMARY KEY REFERENCES workflows(id) ON DELETE CASCADE,
    intake_id uuid NOT NULL,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    factory_id uuid NOT NULL REFERENCES factories(id) ON DELETE CASCADE,
    inventory jsonb NOT NULL DEFAULT '[]'::jsonb,
    retired_at timestamp with time zone NOT NULL
);

COMMIT;
