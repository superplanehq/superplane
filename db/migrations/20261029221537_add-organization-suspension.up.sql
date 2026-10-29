ALTER TABLE organizations
  ADD COLUMN suspended_at timestamptz,
  ADD COLUMN suspended_by_account_id uuid,
  ADD COLUMN suspension_reason text NOT NULL DEFAULT '';

CREATE TABLE organization_suspension_events (
  id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  account_id uuid NOT NULL,
  action text NOT NULL CHECK (action IN ('suspend', 'reactivate')),
  reason text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  cancellation_completed_at timestamptz
);
CREATE INDEX organization_suspension_events_pending_idx
  ON organization_suspension_events (created_at)
  WHERE action = 'suspend' AND cancellation_completed_at IS NULL;
