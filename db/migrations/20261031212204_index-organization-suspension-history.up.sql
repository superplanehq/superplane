CREATE INDEX organization_suspension_events_history_idx
  ON organization_suspension_events (organization_id, created_at DESC, id DESC);
