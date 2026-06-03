# Runner Metrics

Metrics are exported via OpenTelemetry to Dash0.

Pool-scoped metrics carry a `fleet_id` attribute so data can be filtered and
grouped per pool (e.g. `aws-standard-1` vs `aws-arm64-1`).

## Export configuration

**task-broker** and **fleet-manager** read standard OpenTelemetry environment
variables at process startup. When `OTEL_EXPORTER_OTLP_ENDPOINT` is unset,
metrics export is disabled (no-op meter provider; local dev unchanged).

| Variable | Description |
|----------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Dash0 OTLP ingress URL (required to enable export) |
| `OTEL_EXPORTER_OTLP_HEADERS` | Auth header, e.g. `Authorization=Bearer <dash0-token>` |
| `OTEL_SERVICE_NAME` | `task-broker` or `fleet-manager` (appears as `service.name`) |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` or `grpc` (optional; autoexport selects exporter) |
| `OTEL_METRICS_EXPORTER` | Set to `none` to disable metrics while keeping endpoint set |

**task-broker only**

| Variable | Default | Description |
|----------|---------|-------------|
| `METRICS_SAMPLE_INTERVAL_SEC` | `30` | How often to sample `runner.tasks.queued` / `runner.tasks.claimed` gauges |

Fleet-manager pool settings live in the JSON config file (`FM_CONFIG_FILE`); OTLP
export is **not** in that file — set the `OTEL_*` variables on the fleet-manager
process (Docker `--env-file`, systemd `Environment=`, etc.). See
`scripts/deploy/task-broker.env.example` and `scripts/deploy/fleet-manager.env.example`.

Example (task-broker):

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=https://ingress.<region>.aws.dash0.com:4317
export OTEL_EXPORTER_OTLP_HEADERS="Authorization=Bearer <token>"
export OTEL_SERVICE_NAME=task-broker
```

---

## Canonical instrument names

All instruments use the `runner.` prefix so they stay distinct in shared backends
like Dash0. `service.name` (`task-broker` or `fleet-manager`) is set via
`OTEL_SERVICE_NAME`.

| Instrument | Type | Attributes |
|------------|------|------------|
| `runner.tasks.created` | Counter | `fleet_id` |
| `runner.tasks.completed` | Counter | `fleet_id`, `outcome` |
| `runner.task.start_latency` | Histogram (s) | `fleet_id` |
| `runner.tasks.queued` | Gauge | `fleet_id` |
| `runner.tasks.claimed` | Gauge | `fleet_id` |
| `runner.tasks.unclaimed` | Counter | `fleet_id` |
| `runner.lease.reaps` | Counter | `fleet_id` |
| `runner.webhook.deliveries` | Counter | `fleet_id`, `outcome` |
| `runner.webhook.delivery.duration` | Histogram (s) | `fleet_id`, `outcome` |
| `runner.pool.hot_instances` | Gauge | `fleet_id` |
| `runner.instance.spinup.duration` | Histogram (s) | `fleet_id`, `phase` |
| `runner.pool.reconcile.duration` | Histogram (s) | `fleet_id` |

`runner.instance.spinup.duration` is emitted by **both** fleet-manager and
task-broker on the same instrument name. Filter by `phase` (`instance_running`
from fleet-manager, `runner_connected` from task-broker) and/or `service.name`.

`outcome` values: `succeeded`, `failed`, `canceled` (task lifecycle) or
`succeeded`, `failed` (webhook delivery). `phase` values: `instance_running`,
`runner_connected`.

The `runner_connected` spinup phase requires FM-launched runners: cloud-init
sets `RUNNER_LAUNCH_REQUESTED_AT` and the runner forwards it on the WebSocket
hello. Locally started runners (`make runner`) omit it.

---

## Task lifecycle

**Tasks created**
Total number of tasks submitted to the broker.
Attributes: `fleet_id`

**Tasks completed**
Total number of tasks that reached a terminal state, recorded separately by outcome:
`succeeded`, `failed`, or `canceled` (matches broker task status).
Use for success rate and failure/cancel alerting per pool.
Attributes: `fleet_id`, `outcome` (`succeeded` | `failed` | `canceled`)

**Task start latency**
Time from task creation until a runner claims it.
Measures queue wait — the primary SLI for the runner platform (how long work waits for capacity).
Job runtime after claim depends on user commands and is not tracked here.
Attributes: `fleet_id`

**Tasks queued**
Current number of tasks waiting for a runner to claim them (`status = queued`).
Sampled periodically.
Attributes: `fleet_id`

**Tasks claimed**
Current number of tasks claimed by a runner but not yet terminal (`status = claimed`).
Sampled periodically. A sustained backlog here can indicate stuck claims or insufficient runners.
Attributes: `fleet_id`

**Tasks unclaimed**
Total number of tasks that were claimed by a runner but re-queued because
the runner could not start execution (e.g. failed WebSocket delivery).
Attributes: `fleet_id`

**Lease reaps**
Total number of expired task leases reaped. An increase indicates runners
are dying without completing or reporting back.
Attributes: `fleet_id`

---

## Webhook delivery

After a task reaches a terminal state, the broker POSTs the outcome to the
caller-provided `webhook_url` (typically Superplane). These metrics track
whether that outbound notification succeeded — not the health of individual
external endpoints (avoid high-cardinality URL labels).

**Webhook deliveries**
Total number of terminal-state webhook POST attempts that finished (success or
retries exhausted). Use for caller notification reliability alerting.
Attributes: `fleet_id`, `outcome` (`succeeded` | `failed`)

**Webhook delivery duration**
Wall time for one delivery, including backoff between retries.
Attributes: `fleet_id`, `outcome` (`succeeded` | `failed`)

---

## EC2 pool

**Hot instances**
Current number of live EC2 instances in the warm pool.
Sampled periodically.
Attributes: `fleet_id`

**Instance spinup duration**
Time from when an instance is requested until it is ready to accept tasks.
Attributes: `fleet_id`, `phase` (`instance_running` | `runner_connected`)

**Reconcile duration**
Time taken by one fleet-manager reconcile tick.
Attributes: `fleet_id`
