BEGIN;

CREATE TABLE factory_pull_requests (
  id UUID PRIMARY KEY,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
  factory_id UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
  work_order_id UUID NOT NULL REFERENCES factory_work_orders(id) ON DELETE RESTRICT,
  provider TEXT NOT NULL,
  external_id TEXT,
  repository TEXT NOT NULL,
  number BIGINT NOT NULL,
  url TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL,
  merged_at TIMESTAMPTZ,
  closed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT factory_pull_requests_number_positive CHECK (number > 0),
  CONSTRAINT factory_pull_requests_state_valid CHECK (state IN ('open', 'draft', 'closed', 'merged'))
);

CREATE UNIQUE INDEX idx_factory_pull_requests_factory_provider_repo_number
  ON factory_pull_requests (factory_id, provider, repository, number);

CREATE UNIQUE INDEX idx_factory_pull_requests_factory_url
  ON factory_pull_requests (factory_id, url);

CREATE UNIQUE INDEX idx_factory_pull_requests_factory_provider_external
  ON factory_pull_requests (factory_id, provider, external_id)
  WHERE external_id IS NOT NULL;

CREATE INDEX idx_factory_pull_requests_work_order
  ON factory_pull_requests (work_order_id, created_at);

CREATE INDEX idx_factory_pull_requests_factory_state
  ON factory_pull_requests (factory_id, state);

CREATE INDEX idx_factory_pull_requests_factory_merged_at
  ON factory_pull_requests (factory_id, merged_at DESC)
  WHERE merged_at IS NOT NULL;

CREATE INDEX idx_factory_pull_requests_factory_closed_at
  ON factory_pull_requests (factory_id, closed_at DESC)
  WHERE closed_at IS NOT NULL;

CREATE TABLE factory_pull_request_runs (
  pull_request_id UUID NOT NULL
    REFERENCES factory_pull_requests(id) ON DELETE RESTRICT,
  run_id UUID NOT NULL
    REFERENCES workflow_runs(id) ON DELETE RESTRICT,
  description TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (pull_request_id, run_id),
  CONSTRAINT idx_factory_pull_request_runs_run_unique UNIQUE (run_id)
);

DO $$
DECLARE
  total_pr_artifacts BIGINT;
  inserted_pull_requests BIGINT;
  invalid_rows BIGINT;
  duplicate_rows BIGINT;
  migrated_events BIGINT;
  removed_events BIGINT;
  deleted_artifacts BIGINT;
BEGIN
  SELECT COUNT(*) INTO total_pr_artifacts
  FROM factory_work_order_artifacts
  WHERE type = 'pr';

  CREATE TEMP TABLE tmp_pr_candidates ON COMMIT DROP AS
  WITH raw AS (
    SELECT
      a.id,
      a.organization_id,
      a.factory_id,
      a.work_order_id,
      a.data,
      a.key,
      a.merged_at,
      a.closed_at,
      a.created_at,
      regexp_replace(
        regexp_replace(
          btrim(COALESCE(NULLIF(a.data->>'url', ''), COALESCE(a.key, ''))),
          '#.*$',
          ''
        ),
        '/+$',
        ''
      ) AS url
    FROM factory_work_order_artifacts a
    INNER JOIN factory_work_orders wo ON wo.id = a.work_order_id
    WHERE a.type = 'pr'
  ),
  parsed AS (
    SELECT
      raw.*,
      CASE
        WHEN raw.url ~* 'github\.com/' THEN 'github'
        WHEN raw.url ~* 'bitbucket\.org/' THEN 'bitbucket'
        ELSE NULL
      END AS provider,
      COALESCE(
        NULLIF(btrim(raw.data->>'repository'), ''),
        CASE
          WHEN raw.url ~* 'github\.com/[^/]+/[^/]+/pull/' THEN
            (regexp_match(raw.url, 'github\.com/([^/]+/[^/]+)/pull/'))[1]
          WHEN raw.url ~* 'bitbucket\.org/[^/]+/[^/]+/pull-requests/' THEN
            (regexp_match(raw.url, 'bitbucket\.org/([^/]+/[^/]+)/pull-requests/'))[1]
          ELSE NULL
        END
      ) AS repository,
      COALESCE(
        CASE
          WHEN (raw.data->>'number') ~ '^[0-9]+$' THEN (raw.data->>'number')::bigint
          ELSE NULL
        END,
        CASE
          WHEN raw.url ~* 'github\.com/.+/pull/[0-9]+' THEN
            ((regexp_match(raw.url, 'github\.com/.+/pull/([0-9]+)'))[1])::bigint
          WHEN raw.url ~* 'bitbucket\.org/.+/pull-requests/[0-9]+' THEN
            ((regexp_match(raw.url, 'bitbucket\.org/.+/pull-requests/([0-9]+)'))[1])::bigint
          ELSE NULL
        END
      ) AS number,
      COALESCE(btrim(raw.data->>'title'), '') AS title,
      CASE
        WHEN raw.merged_at IS NOT NULL
          OR lower(COALESCE(raw.data->>'state', '')) = 'merged'
          OR lower(COALESCE(raw.data->>'merged', '')) IN ('true', '1')
          THEN 'merged'
        WHEN lower(COALESCE(raw.data->>'state', '')) = 'draft'
          OR lower(COALESCE(raw.data->>'draft', '')) IN ('true', '1')
          THEN 'draft'
        WHEN raw.closed_at IS NOT NULL
          OR lower(COALESCE(raw.data->>'state', '')) = 'closed'
          THEN 'closed'
        ELSE 'open'
      END AS state
    FROM raw
  )
  SELECT
    parsed.*,
    parsed.url ~* '^https?://' AS has_valid_url,
    parsed.provider IS NOT NULL AS has_supported_provider,
    parsed.repository IS NOT NULL AND parsed.repository <> '' AS has_repository,
    parsed.number IS NOT NULL AND parsed.number > 0 AS has_number
  FROM parsed;

  CREATE TEMP TABLE tmp_pr_eligible ON COMMIT DROP AS
  SELECT *
  FROM tmp_pr_candidates
  WHERE has_valid_url
    AND has_supported_provider
    AND has_repository
    AND has_number;

  CREATE TEMP TABLE tmp_pr_identity_conflicts ON COMMIT DROP AS
  SELECT factory_id, provider, repository, number
  FROM tmp_pr_eligible
  GROUP BY factory_id, provider, repository, number
  HAVING COUNT(DISTINCT work_order_id) > 1
  UNION
  SELECT e.factory_id, e.provider, e.repository, e.number
  FROM tmp_pr_eligible e
  WHERE EXISTS (
    SELECT 1
    FROM tmp_pr_eligible other
    WHERE other.factory_id = e.factory_id
      AND other.url = e.url
      AND other.work_order_id <> e.work_order_id
  );

  CREATE TEMP TABLE tmp_pr_insert ON COMMIT DROP AS
  SELECT DISTINCT ON (ranked.factory_id, ranked.url)
    ranked.*
  FROM (
    SELECT DISTINCT ON (e.factory_id, e.provider, e.repository, e.number)
      e.*
    FROM tmp_pr_eligible e
    WHERE NOT EXISTS (
      SELECT 1
      FROM tmp_pr_identity_conflicts c
      WHERE c.factory_id = e.factory_id
        AND c.provider = e.provider
        AND c.repository = e.repository
        AND c.number = e.number
    )
    ORDER BY e.factory_id, e.provider, e.repository, e.number, e.created_at ASC, e.id ASC
  ) ranked
  ORDER BY ranked.factory_id, ranked.url, ranked.created_at ASC, ranked.id ASC;

  INSERT INTO factory_pull_requests (
    id,
    organization_id,
    factory_id,
    work_order_id,
    provider,
    repository,
    number,
    url,
    title,
    state,
    merged_at,
    closed_at,
    created_at,
    updated_at
  )
  SELECT
    gen_random_uuid(),
    organization_id,
    factory_id,
    work_order_id,
    provider,
    repository,
    number,
    url,
    title,
    state,
    merged_at,
    closed_at,
    created_at,
    created_at
  FROM tmp_pr_insert;

  GET DIAGNOSTICS inserted_pull_requests = ROW_COUNT;

  SELECT COUNT(*) INTO invalid_rows
  FROM tmp_pr_candidates
  WHERE NOT (has_valid_url AND has_supported_provider AND has_repository AND has_number);

  SELECT COUNT(*) INTO duplicate_rows
  FROM tmp_pr_eligible e
  WHERE EXISTS (
    SELECT 1
    FROM tmp_pr_identity_conflicts c
    WHERE c.factory_id = e.factory_id
      AND c.provider = e.provider
      AND c.repository = e.repository
      AND c.number = e.number
  );

  WITH matched AS (
    UPDATE factory_work_order_events e
    SET
      type = 'order.pull_request.added',
      data = (e.data - 'artifact') || jsonb_build_object(
        'pullRequest', jsonb_build_object(
          'id', pr.id,
          'provider', pr.provider,
          'repository', pr.repository,
          'number', pr.number,
          'url', pr.url,
          'title', pr.title,
          'state', pr.state
        )
      )
    FROM factory_work_order_artifacts a
    INNER JOIN factory_pull_requests pr
      ON pr.factory_id = a.factory_id
     AND pr.url = regexp_replace(
       regexp_replace(
         btrim(COALESCE(NULLIF(a.data->>'url', ''), COALESCE(a.key, ''))),
         '#.*$',
         ''
       ),
       '/+$',
       ''
     )
    WHERE e.type = 'order.artifact.added'
      AND e.data->'artifact'->>'type' = 'pr'
      AND (e.data->'artifact'->>'id') = a.id::text
    RETURNING e.id
  )
  SELECT COUNT(*) INTO migrated_events FROM matched;

  WITH unmatched AS (
    DELETE FROM factory_work_order_events e
    WHERE e.type = 'order.artifact.added'
      AND e.data->'artifact'->>'type' = 'pr'
    RETURNING e.id
  )
  SELECT COUNT(*) INTO removed_events FROM unmatched;

  DELETE FROM factory_work_order_artifacts
  WHERE type = 'pr';
  GET DIAGNOSTICS deleted_artifacts = ROW_COUNT;

  RAISE NOTICE 'factory_pull_requests backfill: total_pr_artifacts=%, inserted=%, invalid=%, duplicate_identity=%, events_migrated=%, events_removed=%, artifacts_deleted=%',
    total_pr_artifacts,
    inserted_pull_requests,
    invalid_rows,
    duplicate_rows,
    migrated_events,
    removed_events,
    deleted_artifacts;
END $$;

DROP INDEX IF EXISTS idx_factory_work_order_artifacts_pr_closed_at;
DROP INDEX IF EXISTS idx_factory_work_order_artifacts_pr_merged_at;

ALTER TABLE factory_work_order_artifacts
  DROP COLUMN IF EXISTS merged_at,
  DROP COLUMN IF EXISTS closed_at;

COMMIT;
