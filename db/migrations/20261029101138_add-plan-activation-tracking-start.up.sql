-- Earlier activations are not tracked. The cutoff is the moment this migration
-- runs. Writers that record activations must deploy with this migration.
-- The row lives outside installation_metadata, which the installation
-- singleton can recreate or reset, so the cutoff never changes after this run.
ALTER TABLE installation_metadata DROP COLUMN IF EXISTS plan_activation_tracking_started_at;

CREATE TABLE IF NOT EXISTS plan_activation_tracking (
    id integer PRIMARY KEY CHECK (id = 1),
    started_at timestamp with time zone NOT NULL
);

INSERT INTO plan_activation_tracking (id, started_at)
VALUES (1, now())
ON CONFLICT (id) DO NOTHING;
