#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

# Local only. Inserts fake hosted model and runner-VM usage into
# superplane_dev so Spending charts have data without real runs.

DB_NAME="${1:-superplane_dev}"
MONEY="${MONEY:-}"
TASKS="${TASKS:-20}"
DAYS="${DAYS:-30}"
ORGANIZATION_ID="${ORGANIZATION_ID:-}"
FACTORY_ID="${FACTORY_ID:-}"

export PGPASSWORD=the-cake-is-a-lie

usage() {
  echo "Usage: make simulate.usage MONEY=20" >&2
  echo "Optional: TASKS=20 DAYS=30 ORGANIZATION_ID=<uuid> FACTORY_ID=<uuid>" >&2
}

if [[ "$DB_NAME" != "superplane_dev" ]]; then
  echo "simulate.usage only runs against superplane_dev." >&2
  exit 1
fi

if [[ -z "$MONEY" ]]; then
  echo "MONEY is required. Example: make simulate.usage MONEY=20" >&2
  usage
  exit 1
fi

if ! [[ "$MONEY" =~ ^[0-9]+([.][0-9]{1,2})?$ ]]; then
  echo "MONEY must be a dollar amount such as 20 or 20.50." >&2
  usage
  exit 1
fi

if [[ "$MONEY" == *.* ]]; then
  money_dollars="${MONEY%%.*}"
  money_cents="${MONEY#*.}00"
  money_cents="${money_cents:0:2}"
  MONEY_CENTS=$((10#${money_dollars:-0} * 100 + 10#$money_cents))
else
  MONEY_CENTS=$((10#$MONEY * 100))
fi

if (( MONEY_CENTS < 1 )); then
  echo "MONEY must be at least 0.01." >&2
  usage
  exit 1
fi

if (( MONEY_CENTS > 10000000 )); then
  echo "MONEY must be at most 100000." >&2
  usage
  exit 1
fi

MONEY_MICROS=$((MONEY_CENTS * 10000))

if ! [[ "$TASKS" =~ ^[1-9][0-9]*$ ]] || (( TASKS > 1000 )); then
  echo "TASKS must be an integer from 1 to 1000." >&2
  usage
  exit 1
fi

if (( MONEY_MICROS < TASKS * 2 )); then
  echo "MONEY is too small to spread across ${TASKS} tasks. Use fewer TASKS." >&2
  exit 1
fi

if ! [[ "$DAYS" =~ ^[1-9][0-9]*$ ]] || (( DAYS > 366 )); then
  echo "DAYS must be an integer from 1 to 366." >&2
  usage
  exit 1
fi

uuid_re='^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$'

if [[ -n "$ORGANIZATION_ID" && ! "$ORGANIZATION_ID" =~ $uuid_re ]]; then
  echo "ORGANIZATION_ID must be a UUID." >&2
  exit 1
fi

if [[ -n "$FACTORY_ID" && ! "$FACTORY_ID" =~ $uuid_re ]]; then
  echo "FACTORY_ID must be a UUID." >&2
  exit 1
fi

psql_dev() {
  psql -h db -p 5432 -U postgres -d "$DB_NAME" -v ON_ERROR_STOP=1 "$@"
}

if [[ -n "$ORGANIZATION_ID" ]]; then
  ORG_ROW="$(psql_dev -tA -c "
    SELECT id::text || '|' || name
    FROM organizations
    WHERE id = '${ORGANIZATION_ID}'::uuid
      AND deleted_at IS NULL
  ")"
  if [[ -z "$ORG_ROW" ]]; then
    echo "No organization ${ORGANIZATION_ID} in superplane_dev." >&2
    exit 1
  fi
else
  ORG_ROW="$(psql_dev -tA -c "
    SELECT id::text || '|' || name
    FROM organizations
    WHERE deleted_at IS NULL
    ORDER BY created_at ASC NULLS LAST, id ASC
    LIMIT 1
  ")"
  if [[ -z "$ORG_ROW" ]]; then
    echo "No organization in superplane_dev. Create one in the UI first." >&2
    exit 1
  fi
fi

ORG_ID="${ORG_ROW%%|*}"
ORG_NAME="${ORG_ROW#*|}"

FACTORY_FILTER="organization_id = '${ORG_ID}'::uuid AND deleted_at IS NULL"
if [[ -n "$FACTORY_ID" ]]; then
  FACTORY_FILTER="${FACTORY_FILTER} AND id = '${FACTORY_ID}'::uuid"
fi

FACTORY_COUNT="$(psql_dev -tA -c "
  SELECT COUNT(*) FROM factories WHERE ${FACTORY_FILTER}
")"
if [[ "$FACTORY_COUNT" -eq 0 ]]; then
  if [[ -n "$FACTORY_ID" ]]; then
    echo "No factory ${FACTORY_ID} for organization ${ORG_NAME}." >&2
  else
    echo "No factory for organization ${ORG_NAME}. Create one in the UI first." >&2
  fi
  exit 1
fi

psql_dev \
  -v org_id="$ORG_ID" \
  -v task_count="$TASKS" \
  -v day_count="$DAYS" \
  -v money_micros="$MONEY_MICROS" \
  -v factory_id="${FACTORY_ID:-none}" \
  <<'SQL'
BEGIN;

CREATE TEMP TABLE sim_factories AS
SELECT
  id,
  name,
  row_number() OVER (ORDER BY created_at ASC, id ASC) AS n
FROM factories
WHERE organization_id = :'org_id'::uuid
  AND deleted_at IS NULL
  AND (
    :'factory_id' = 'none'
    OR id = NULLIF(:'factory_id', 'none')::uuid
  );

CREATE TEMP TABLE sim_work_orders AS
SELECT DISTINCT ON (factory_id)
  factory_id,
  id AS work_order_id
FROM factory_work_orders
WHERE factory_id IN (SELECT id FROM sim_factories)
ORDER BY factory_id, created_at DESC, id DESC;

-- Catalog rates match pkg/usage/pricebook (input cents per million tokens,
-- VM micros per second). Local fleet is never used here so VM cost is > 0.
CREATE TEMP TABLE sim_tasks AS
SELECT
  gs AS i,
  f.id AS factory_id,
  wo.work_order_id,
  uuid_generate_v4() AS canvas_run_id,
  uuid_generate_v4() AS model_node_execution_id,
  uuid_generate_v4() AS compute_node_execution_id,
  now() - ((random() * :'day_count'::int) * interval '1 day') AS occurred_at,
  (0.4 + random()) AS weight,
  (55 + floor(random() * 26)::int) AS model_share_pct,
  (ARRAY['anthropic', 'openai', 'openai', 'anthropic'])[1 + ((gs - 1) % 4)] AS provider,
  (ARRAY['claude-sonnet-4-6', 'gpt-4o', 'gpt-5', 'claude-opus-4-6'])[1 + ((gs - 1) % 4)] AS model,
  (ARRAY[300, 250, 125, 1500])[1 + ((gs - 1) % 4)] AS input_cents_per_million,
  CASE
    WHEN gs % 5 = 0 THEN 'e1-tiny-amd64'
    WHEN gs % 3 = 0 THEN 'e1-large-arm64'
    ELSE 'e1-large-amd64'
  END AS machine_type,
  CASE
    WHEN gs % 5 = 0 THEN 3
    WHEN gs % 3 = 0 THEN 50
    ELSE 70
  END AS micros_per_second
FROM generate_series(1, :'task_count'::int) AS gs
JOIN sim_factories f
  ON f.n = 1 + ((gs - 1) % (SELECT COUNT(*) FROM sim_factories)::int)
LEFT JOIN sim_work_orders wo ON wo.factory_id = f.id;

CREATE TEMP TABLE sim_shares AS
SELECT
  i,
  factory_id,
  work_order_id,
  canvas_run_id,
  model_node_execution_id,
  compute_node_execution_id,
  occurred_at,
  provider,
  model,
  machine_type,
  model_share_pct,
  input_cents_per_million,
  micros_per_second,
  floor(
    ((:'money_micros'::bigint) - (2 * :'task_count'::int))
    * weight
    / SUM(weight) OVER ()
  )::bigint AS extra_micros,
  (
    ((:'money_micros'::bigint) - (2 * :'task_count'::int))
    * weight
    / SUM(weight) OVER ()
  ) - floor(
    ((:'money_micros'::bigint) - (2 * :'task_count'::int))
    * weight
    / SUM(weight) OVER ()
  ) AS frac
FROM sim_tasks;

CREATE TEMP TABLE sim_costed AS
SELECT
  i,
  factory_id,
  work_order_id,
  canvas_run_id,
  model_node_execution_id,
  compute_node_execution_id,
  occurred_at,
  provider,
  model,
  machine_type,
  model_share_pct,
  input_cents_per_million,
  micros_per_second,
  2 + extra_micros + CASE
    WHEN row_number() OVER (ORDER BY frac DESC, i ASC)
      <= (
        (:'money_micros'::bigint) - (2 * :'task_count'::int)
        - SUM(extra_micros) OVER ()
      )
    THEN 1
    ELSE 0
  END AS cost_micros
FROM sim_shares;

CREATE TEMP TABLE sim_priced AS
SELECT
  i,
  factory_id,
  work_order_id,
  canvas_run_id,
  model_node_execution_id,
  compute_node_execution_id,
  occurred_at,
  provider,
  model,
  machine_type,
  LEAST(
    cost_micros - 1,
    GREATEST(1, (cost_micros * model_share_pct) / 100)
  ) AS model_cost_micros,
  cost_micros - LEAST(
    cost_micros - 1,
    GREATEST(1, (cost_micros * model_share_pct) / 100)
  ) AS compute_cost_micros,
  input_cents_per_million,
  micros_per_second
FROM sim_costed;

INSERT INTO workspace_usage_events (
  id,
  organization_id,
  factory_id,
  work_order_id,
  canvas_run_id,
  node_execution_id,
  node_id,
  provider,
  model,
  usage_kind,
  funding_source,
  input_tokens,
  output_tokens,
  cache_read_tokens,
  cache_write_tokens,
  reasoning_tokens,
  total_tokens,
  duration_seconds,
  machine_type,
  fleet_id,
  cost_micros,
  provider_cost_micros,
  currency,
  price_book_version,
  idempotency_key,
  occurred_at,
  created_at
)
SELECT
  uuid_generate_v4(),
  :'org_id'::uuid,
  factory_id,
  work_order_id,
  canvas_run_id,
  model_node_execution_id,
  'prompt',
  provider,
  model,
  'model',
  'hosted',
  GREATEST(1, (model_cost_micros * 100) / input_cents_per_million),
  0,
  0,
  0,
  0,
  GREATEST(1, (model_cost_micros * 100) / input_cents_per_million),
  0,
  '',
  '',
  model_cost_micros,
  model_cost_micros,
  'usd',
  '2026-08-19.2',
  'simulate:usage:model:' || canvas_run_id::text,
  occurred_at,
  occurred_at
FROM sim_priced;

INSERT INTO workspace_usage_events (
  id,
  organization_id,
  factory_id,
  work_order_id,
  canvas_run_id,
  node_execution_id,
  node_id,
  provider,
  model,
  usage_kind,
  funding_source,
  input_tokens,
  output_tokens,
  cache_read_tokens,
  cache_write_tokens,
  reasoning_tokens,
  total_tokens,
  duration_seconds,
  machine_type,
  fleet_id,
  cost_micros,
  provider_cost_micros,
  currency,
  price_book_version,
  idempotency_key,
  occurred_at,
  created_at
)
SELECT
  uuid_generate_v4(),
  :'org_id'::uuid,
  factory_id,
  work_order_id,
  canvas_run_id,
  compute_node_execution_id,
  'runner',
  'runner',
  machine_type,
  'compute',
  'hosted',
  0,
  0,
  0,
  0,
  0,
  0,
  GREATEST(1, compute_cost_micros / micros_per_second),
  machine_type,
  machine_type,
  compute_cost_micros,
  compute_cost_micros,
  'usd',
  '2026-08-19.2',
  'simulate:usage:compute:' || canvas_run_id::text,
  occurred_at,
  occurred_at
FROM sim_priced;

COMMIT;

SELECT
  :'task_count'::int AS tasks,
  COUNT(*) FILTER (WHERE usage_kind = 'model') AS model_events,
  COUNT(*) FILTER (WHERE usage_kind = 'compute') AS compute_events,
  ROUND(SUM(cost_micros) FILTER (WHERE usage_kind = 'model') / 1000000.0, 2) AS model_usd,
  ROUND(SUM(cost_micros) FILTER (WHERE usage_kind = 'compute') / 1000000.0, 2) AS compute_usd,
  ROUND(SUM(cost_micros) / 1000000.0, 2) AS total_usd
FROM workspace_usage_events
WHERE canvas_run_id IN (SELECT canvas_run_id FROM sim_priced);

SELECT id::text AS factory_id, name AS factory
FROM sim_factories
ORDER BY n;
SQL

echo "Organization: ${ORG_NAME} (${ORG_ID})"
echo "Spread \$${MONEY} across ${TASKS} tasks over ${DAYS} days (model tokens plus runner VM time)."
echo "Open Organization settings → Spending to view the rows."
