CREATE TABLE plan_feature_definitions (
 key text PRIMARY KEY CHECK (key ~ '^[a-z][a-z0-9_]*$'),
 label text NOT NULL CHECK (btrim(label) <> ''),
 gated boolean NOT NULL DEFAULT true
);

INSERT INTO plan_feature_definitions (key, label, gated) VALUES
 ('byo_subscription', 'Bring your own subscription', false),
 ('byo_key', 'Bring your own API key', false),
 ('centralized_billing', 'Centralized billing', false),
 ('self_hosted_app', 'Self-hosted application', false),
 ('priority_support', 'Priority support', false);
