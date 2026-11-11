\set ON_ERROR_STOP on
\pset pager off
-- Session-local fixtures never change application data or indexes.
SET timezone = 'UTC';
CREATE TEMP TABLE organizations (LIKE public.organizations INCLUDING ALL);
CREATE TEMP TABLE workflows (LIKE public.workflows INCLUDING ALL);
CREATE TEMP TABLE workflow_runs (LIKE public.workflow_runs INCLUDING DEFAULTS INCLUDING CONSTRAINTS);
CREATE INDEX ON workflow_runs (workflow_id, created_at DESC);
CREATE INDEX ON workflow_runs (workflow_id, state);
CREATE INDEX ON workflow_runs (workflow_id, node_id);
CREATE INDEX ON workflow_runs (version_id);
CREATE INDEX ON workflow_runs (parent_run_id) INCLUDE (id) WHERE parent_run_id IS NOT NULL;
CREATE INDEX ON workflow_runs (parent_workflow_id, parent_execution_id) WHERE parent_execution_id IS NOT NULL;
CREATE INDEX ON workflow_runs (cancelled_at) WHERE state = 'cancelling';
ALTER TABLE workflow_runs ADD PRIMARY KEY (id);
CREATE TEMP TABLE workflow_node_executions (LIKE public.workflow_node_executions INCLUDING ALL);
CREATE TEMP TABLE workflow_node_queue_items (LIKE public.workflow_node_queue_items INCLUDING ALL);
CREATE TEMP TABLE workflow_node_requests (LIKE public.workflow_node_requests INCLUDING ALL);
INSERT INTO organizations (id, name, slug, usage_retention_window_days)
SELECT md5('org' || n)::uuid, 'retention-' || n, 'retention-' || n,
       CASE n % 5 WHEN 0 THEN NULL WHEN 1 THEN 0 WHEN 2 THEN 7 WHEN 3 THEN 30 ELSE 90 END
FROM generate_series(1, 100) n;
INSERT INTO workflows (id, organization_id, name, live_version_id, created_at, updated_at)
SELECT md5('workflow' || n)::uuid, md5('org' || (1 + n % 100))::uuid,
       'retention-' || n, md5('version' || n)::uuid, '2026-11-01', '2026-11-01'
FROM generate_series(1, 1000) n;
INSERT INTO workflow_runs (id, workflow_id, state, version_id, created_at, updated_at, finished_at, input)
SELECT md5('run' || n)::uuid, md5('workflow' || (1 + n % 1000))::uuid,
       CASE WHEN n % 20 = 0 THEN 'started' ELSE 'finished' END,
       md5('version' || (1 + n % 1000))::uuid, '2026-07-01', '2026-11-01',
       timestamp '2026-11-01 12:00:00' - ((n * 7919 % 12000000) * interval '1 second'),
       jsonb_build_object('payload', repeat('x', 512))
FROM generate_series(1, 200000) n;
INSERT INTO workflow_node_executions (id, workflow_id, node_id, run_id, state, created_at, updated_at)
SELECT md5('execution' || id)::uuid, workflow_id, 'node', id,
       CASE WHEN row_number() OVER (ORDER BY id) % 100 = 0 THEN 'started' ELSE 'finished' END,
       created_at, updated_at FROM workflow_runs;
INSERT INTO workflow_node_queue_items (workflow_id, node_id, run_id, created_at)
SELECT workflow_id, node_id, run_id, created_at FROM workflow_node_executions WHERE state = 'started';
INSERT INTO workflow_node_requests (workflow_id, node_id, execution_id, state, type, spec, run_at, created_at, updated_at)
SELECT workflow_id, node_id, id, CASE WHEN row_number() OVER (ORDER BY id) % 200 = 0 THEN 'pending' ELSE 'completed' END,
       'invoke_action', '{}', created_at, created_at, updated_at FROM workflow_node_executions;
ANALYZE organizations;
ANALYZE workflows;
ANALYZE workflow_node_executions;
ANALYZE workflow_node_queue_items;
ANALYZE workflow_node_requests;
CREATE TEMP TABLE retention_measurements (dataset text, variant text, plan_mode text, iteration int, plan jsonb);
CREATE TEMP TABLE retention_expected (dataset text, ids uuid[]);
DO $comparison$
DECLARE
    dataset text;
    variant text;
    mode text;
    iteration int;
    query text;
    predicate text;
    plan jsonb;
    actual uuid[];
    expected uuid[];
BEGIN
    FOREACH dataset IN ARRAY ARRAY['expired batch', 'recent empty'] LOOP
        IF dataset = 'recent empty' THEN
            CREATE TEMP TABLE recent_runs AS SELECT * FROM workflow_runs;
            UPDATE recent_runs SET finished_at = timestamp '2026-10-31 12:00:00';
            TRUNCATE workflow_runs;
            INSERT INTO workflow_runs SELECT * FROM recent_runs;
            DROP TABLE recent_runs;
            DROP INDEX retention_finished_at;
        END IF;
        ANALYZE workflow_runs;
        FOREACH variant IN ARRAY ARRAY['baseline', 'ordered index', 'index and rewrite', 'index and minimum window', 'minimum window and literal state'] LOOP
            IF variant = 'ordered index' THEN
                CREATE INDEX retention_finished_at ON workflow_runs (finished_at)
                WHERE state = 'finished' AND finished_at IS NOT NULL;
                ANALYZE workflow_runs;
            END IF;
            predicate := 'workflow_runs.finished_at + organizations.usage_retention_window_days * INTERVAL ''1 day'' < $2';
            IF variant = 'index and rewrite' THEN
                predicate := 'workflow_runs.finished_at < $2 - organizations.usage_retention_window_days * INTERVAL ''1 day''';
            END IF;
            IF variant IN ('index and minimum window', 'minimum window and literal state') THEN
                predicate := predicate || ' AND workflow_runs.finished_at < (SELECT $2 - min(usage_retention_window_days) * INTERVAL ''1 day'' FROM organizations WHERE usage_retention_window_days > 0)';
            END IF;
            query := 'SELECT workflow_runs.* FROM workflow_runs
                JOIN workflows ON workflow_runs.workflow_id = workflows.id
                JOIN organizations ON workflows.organization_id = organizations.id
                WHERE organizations.usage_retention_window_days IS NOT NULL
                AND organizations.usage_retention_window_days > 0
                AND workflow_runs.state = $1 AND workflow_runs.finished_at IS NOT NULL AND ' || predicate || '
                AND NOT EXISTS (SELECT 1 FROM workflow_node_queue_items WHERE run_id = workflow_runs.id)
                AND NOT EXISTS (SELECT 1 FROM workflow_node_executions WHERE run_id = workflow_runs.id AND state IN ($3, $4, $5))
                AND NOT EXISTS (SELECT 1 FROM workflow_node_requests JOIN workflow_node_executions ON execution_id = workflow_node_executions.id
                    WHERE workflow_node_executions.run_id = workflow_runs.id AND workflow_node_requests.state = $6)
                ORDER BY workflow_runs.finished_at ASC LIMIT $7';
            IF variant = 'minimum window and literal state' THEN
                query := replace(query, 'workflow_runs.state = $1', 'workflow_runs.state = ''finished''');
            END IF;
            -- Compare all eligible IDs, independent of ties at the batch boundary.
            EXECUTE 'SELECT array_agg(id ORDER BY id) FROM (' || replace(query, ' LIMIT $7', '') || ') candidates'
                INTO actual USING 'finished', timestamp '2026-11-01 12:00:00', 'pending', 'started', 'cancelling', 'pending';
            IF variant = 'baseline' THEN
                expected := actual;
                INSERT INTO retention_expected VALUES (dataset, expected);
            ELSIF actual IS DISTINCT FROM expected THEN
                RAISE EXCEPTION 'Eligibility changed for % / %', dataset, variant;
            END IF;
            EXECUTE 'PREPARE retention_selection (text, timestamp, text, text, text, text, int) AS ' || query;
            FOREACH mode IN ARRAY ARRAY['force_custom_plan', 'force_generic_plan'] LOOP
                PERFORM set_config('plan_cache_mode', mode, false);
                FOR iteration IN 1..7 LOOP
                    EXECUTE 'EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) EXECUTE retention_selection(''finished'', ''2026-11-01 12:00:00'', ''pending'', ''started'', ''cancelling'', ''pending'', 1000)' INTO plan;
                    INSERT INTO retention_measurements VALUES (dataset, variant, mode, iteration, plan);
                END LOOP;
            END LOOP;
            DEALLOCATE retention_selection;
        END LOOP;
    END LOOP;
END
$comparison$;
SELECT version(), current_setting('shared_buffers') AS shared_buffers,
       current_setting('work_mem') AS work_mem, current_setting('random_page_cost') AS random_page_cost;
SELECT dataset, coalesce(cardinality(ids), 0) AS eligible_runs FROM retention_expected;
SELECT dataset, variant, plan_mode, count(*) AS samples,
       round(percentile_cont(0.5) WITHIN GROUP (ORDER BY (plan->0->>'Execution Time')::numeric)::numeric, 3) AS median_ms,
       percentile_cont(0.5) WITHIN GROUP (ORDER BY (plan->0->'Plan'->>'Local Hit Blocks')::numeric) AS median_local_hits,
       percentile_cont(0.5) WITHIN GROUP (ORDER BY (plan->0->'Plan'->>'Local Read Blocks')::numeric) AS median_local_reads,
       percentile_cont(0.5) WITHIN GROUP (ORDER BY (plan->0->'Plan'->>'Temp Written Blocks')::numeric) AS median_temp_writes
FROM retention_measurements GROUP BY dataset, variant, plan_mode ORDER BY dataset, variant, plan_mode;
-- One complete plan per variant; all measured plans remain in the session table.
\pset format unaligned
SELECT dataset, variant, plan_mode, plan FROM retention_measurements WHERE iteration = 7;
