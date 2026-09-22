# Performance KPIs

Use these definitions for Dash0 dashboards, check rules, and the morning report.
Do not mix long-poll hold time with interactive API latency.

HTTP metrics include `http.route` and `superplane.request.kind`.
The kind values are `api`, `long_poll`, and `webhook`.

## Interactive API p90

Measure HTTP server request duration for `superplane.request.kind="api"`.

- Exclude `long_poll`.
- Keep webhooks in a separate panel. Do not use them in the interactive p90.
- Ignore series with an empty `http.route`.
- Warn when p90 stays above 500 ms for 10 minutes.
- Alert when p90 stays above 1 s for 10 minutes.

Example PromQL:

```promql
histogram_quantile(
  0.9,
  sum by (le) (
    rate(http_server_request_duration_seconds_bucket{
      superplane_request_kind="api",
      http_route!=""
    }[5m])
  )
)
```

Confirm the exact OTel metric name in Dash0. otelmux records `http.server.request.duration`.

## Server errors

Count HTTP responses with status 500-599.

- Group by `http.route`.
- Exclude client cancel (HTTP 499) and timeouts (HTTP 408).
- Track `long_poll` 5xx on a separate panel.
- Alert when the interactive 5xx rate stays above 0.1% for 15 minutes.

## Long-poll health

Route: `/api/v1/runner/planning-sessions/wait`.

Track these signals, not latency SLOs:

- request count
- hold completions with status `pending`, `message`, `ended`
- 5xx count
- database queries per wait second

A 45 s hold is expected. Do not include this route in interactive API p90.

## Saturation

Correlate API CPU and memory with:

- `db.pool.wait.count`
- `db.locks.count`
- `db.long_queries.count`
- `workflow_events.pending.count`
- `workflow_node_executions.pending.count`

Stable host CPU for PostgreSQL does not mean the API pool is healthy.

## Morning report

The morning report must print:

- Interactive API p90 for last day and last week
- Interactive HTTP 5xx count, grouped by route
- Slowest `api` route
- Long-poll wait count and wait 5xx, on a separate line
- DB CPU, DB memory, DB pool waits

Update the live Performance KPIs canvas after you deploy the `superplane.request.kind` attribute.

## Follow-up: asynchronous webhooks

Keep webhook HTTP handlers short in a later change:

1. Persist the bounded delivery.
2. Return 200 or 202 quickly.
3. Process nodes in a worker with an idempotency key.

Do not combine that behavior change with the telemetry classification change.

## Deferred: GitHub metadata and repository sync

Do not change GitHub metadata lookup or repository synchronization in this
telemetry patch.

Wait until interactive API p90 can exclude long polls. Then confirm that
GitHub list or sync routes stay in the slowest-route list before you add
an index or defer the sync work.
