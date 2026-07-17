begin;

CREATE TABLE canvas_subscriptions (
  source_canvas_id uuid NOT NULL,
  target_canvas_id uuid NOT NULL,
  target_node_id   CHARACTER VARYING(128) NOT NULL,

  PRIMARY KEY (source_canvas_id, target_canvas_id, target_node_id),
  FOREIGN KEY (source_canvas_id) REFERENCES workflows(id) ON DELETE CASCADE,
  FOREIGN KEY (target_canvas_id) REFERENCES workflows(id) ON DELETE CASCADE,
  FOREIGN KEY (target_canvas_id, target_node_id) REFERENCES workflow_nodes(workflow_id, node_id) ON DELETE CASCADE
);

CREATE INDEX idx_canvas_subscriptions_target ON canvas_subscriptions(target_canvas_id, target_node_id);

CREATE TABLE app_messages (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  canvas_id  uuid NOT NULL,
  node_id    CHARACTER VARYING(128) NOT NULL,
  payload    jsonb NOT NULL,
  created_at timestamp NOT NULL DEFAULT now(),

  FOREIGN KEY (canvas_id) REFERENCES workflows(id) ON DELETE CASCADE,
  FOREIGN KEY (canvas_id, node_id) REFERENCES workflow_nodes(workflow_id, node_id) ON DELETE CASCADE
);

CREATE INDEX idx_app_messages_created_at ON app_messages(created_at);

commit;
