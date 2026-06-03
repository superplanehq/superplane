# Runner Metrics

Metrics are exported via OpenTelemetry to Dash0.

Pool-scoped metrics carry a `fleet_id` attribute so data can be filtered and
grouped per pool (e.g. `aws-standard-1` vs `aws-arm64-1`).

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
