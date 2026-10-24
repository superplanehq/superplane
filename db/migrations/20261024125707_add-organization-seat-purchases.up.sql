CREATE TABLE organization_seat_purchases (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    polar_subscription_id text NOT NULL,
    from_seats integer NOT NULL CHECK (from_seats >= 0),
    to_seats integer NOT NULL,
    status text NOT NULL,
    error_code text NOT NULL DEFAULT '',
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT organization_seat_purchases_status
        CHECK (status IN ('pending', 'succeeded', 'failed', 'action_required', 'unknown')),
    CONSTRAINT organization_seat_purchases_adds_seats CHECK (to_seats > from_seats)
);

CREATE INDEX idx_organization_seat_purchases_organization_created
    ON organization_seat_purchases (organization_id, created_at);

CREATE UNIQUE INDEX uq_organization_seat_purchases_in_flight
    ON organization_seat_purchases (organization_id)
    WHERE status IN ('pending', 'unknown');
