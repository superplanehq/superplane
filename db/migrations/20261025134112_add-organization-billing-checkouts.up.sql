CREATE TABLE organization_billing_checkouts (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    plan_key text NOT NULL,
    polar_product_id text NOT NULL,
    seats integer CHECK (seats >= 1),
    state text NOT NULL,
    polar_checkout_id text,
    checkout_url text,
    polar_status text,
    expires_at timestamp with time zone,
    last_error text,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    resolved_at timestamp with time zone,
    CONSTRAINT organization_billing_checkouts_state
        CHECK (state IN ('creating', 'ambiguous', 'open', 'succeeded', 'expired', 'failed', 'released')),
    CONSTRAINT organization_billing_checkouts_resolved_at
        CHECK ((state IN ('creating', 'ambiguous', 'open')) = (resolved_at IS NULL))
);

CREATE INDEX idx_organization_billing_checkouts_organization_created
    ON organization_billing_checkouts (organization_id, created_at);

CREATE UNIQUE INDEX uq_organization_billing_checkouts_unresolved
    ON organization_billing_checkouts (organization_id)
    WHERE state IN ('creating', 'ambiguous', 'open');

CREATE UNIQUE INDEX uq_organization_billing_checkouts_polar_checkout
    ON organization_billing_checkouts (polar_checkout_id)
    WHERE polar_checkout_id IS NOT NULL;
