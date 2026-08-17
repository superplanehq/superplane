BEGIN;

--
-- Per-user emoji reactions on work order comments (`order.comment.added`
-- events). Comments are not a first-class table — they are
-- `factory_work_order_events` rows — so `comment_id` references that
-- table's own primary key, which doubles as the public comment id.
--
CREATE TABLE factory_work_order_comment_reactions (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL,
  factory_id      UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
  work_order_id   UUID NOT NULL REFERENCES factory_work_orders(id) ON DELETE RESTRICT,
  comment_id      UUID NOT NULL REFERENCES factory_work_order_events(id) ON DELETE RESTRICT,
  emoji           VARCHAR(16) NOT NULL,
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One reaction per (comment, user, emoji) — toggling the same emoji twice
-- is idempotent at the model layer, but the constraint is the source of
-- truth that makes concurrent double-clicks safe.
CREATE UNIQUE INDEX idx_factory_work_order_comment_reactions_unique
  ON factory_work_order_comment_reactions (comment_id, user_id, emoji);

-- Backs the per-comment summary aggregation query (ListWorkOrderEvents).
CREATE INDEX idx_factory_work_order_comment_reactions_comment
  ON factory_work_order_comment_reactions (comment_id);

COMMIT;
