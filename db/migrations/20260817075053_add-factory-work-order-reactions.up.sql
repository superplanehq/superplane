CREATE TABLE factory_work_order_reactions (
  work_order_id UUID NOT NULL REFERENCES factory_work_orders(id) ON DELETE RESTRICT,
  user_id       UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  content       VARCHAR(16) NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  PRIMARY KEY (work_order_id, user_id, content)
);

CREATE INDEX idx_factory_work_order_reactions_work_order_id
  ON factory_work_order_reactions (work_order_id);
