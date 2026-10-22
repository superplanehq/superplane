CREATE TABLE organization_seat_limit_overrides (
    organization_id uuid PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    seat_limit integer CHECK (seat_limit > 0)
);
