CREATE TABLE IF NOT EXISTS organization_plan_activations (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    plan_key text NOT NULL,
    activated_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS organization_plan_activations_activated_at_idx
    ON organization_plan_activations (activated_at);
