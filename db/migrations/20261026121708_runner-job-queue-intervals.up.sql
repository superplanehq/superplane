CREATE TABLE runner_task_queue_intervals (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id uuid NOT NULL REFERENCES runner_tasks(id) ON DELETE CASCADE,
    started_at timestamptz NOT NULL,
    ended_at timestamptz,
    CHECK (ended_at IS NULL OR ended_at >= started_at)
);
CREATE UNIQUE INDEX runner_task_queue_intervals_open
    ON runner_task_queue_intervals(task_id) WHERE ended_at IS NULL;
CREATE INDEX runner_task_queue_intervals_task ON runner_task_queue_intervals(task_id, started_at);

-- Capture every transition in the job transaction. Only new invocations opt
-- into queue-aware timeouts, so old invocations need no history backfill.
CREATE FUNCTION record_runner_task_queue_interval() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    was_queued boolean := false;
    is_queued boolean;
    transition_at timestamptz;
BEGIN
    IF TG_OP = 'UPDATE' THEN
        was_queued := OLD.backend = 'integrated' AND OLD.state = 'queued' AND OLD.runner_id IS NULL;
    END IF;
    is_queued := NEW.backend = 'integrated' AND NEW.state = 'queued' AND NEW.runner_id IS NULL;
    transition_at := COALESCE(NEW.updated_at, statement_timestamp());
    IF is_queued AND NOT was_queued THEN
        INSERT INTO runner_task_queue_intervals(task_id, started_at)
        VALUES (NEW.id, transition_at) ON CONFLICT DO NOTHING;
    ELSIF was_queued AND NOT is_queued THEN
        UPDATE runner_task_queue_intervals
        SET ended_at = GREATEST(started_at, transition_at)
        WHERE task_id = NEW.id AND ended_at IS NULL;
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER runner_task_queue_history
AFTER INSERT OR UPDATE OF state, runner_id ON runner_tasks
FOR EACH ROW EXECUTE FUNCTION record_runner_task_queue_interval();
