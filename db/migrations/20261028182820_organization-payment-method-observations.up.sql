CREATE TABLE organization_payment_method_observations (
    organization_id uuid PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    usable boolean NOT NULL,
    read_started_at bigint NOT NULL,
    observed_at timestamptz NOT NULL,
    usable_method_hashes text[] NOT NULL DEFAULT '{}',
    transition_id uuid
);
