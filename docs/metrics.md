# Runner Metrics

Metrics are exported via OpenTelemetry to Dash0.

All metrics that relate to a specific runner pool carry a `fleet_id` attribute
so data can be filtered and grouped per pool (e.g. `aws-standard-1` vs `aws-arm64-1`).

---

## Task lifecycle

**Tasks created**
Total number of tasks submitted to the broker.
Attributes: `fleet_id`

**Tasks completed**
Total number of tasks that reached a terminal state.
Attributes: `fleet_id`, `outcome` (`success` | `failed` | `canceled`)

**Task start latency**
Time from task creation until a runner claims it.
Measures queue wait — the primary SLI for the system.
Attributes: `fleet_id`

**Task duration**
Wall time from task creation to completion.
Attributes: `fleet_id`, `outcome`

**Tasks pending**
Current number of tasks in a queued or claimed state.
Sampled periodically.
Attributes: `fleet_id`

**Tasks unclaimed**
Total number of tasks that were claimed by a runner but re-queued because
the runner could not start execution (e.g. failed WebSocket delivery).
Attributes: `fleet_id`

**Lease reaps**
Total number of expired task leases reaped. An increase indicates runners
are dying without completing or reporting back.

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
