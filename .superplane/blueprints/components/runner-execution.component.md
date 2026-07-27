# Runner Execution Component Blueprint

## Capability Summary

Runner execution lets workflow actions submit scripts or commands to an
external task broker/fleet, inject literal and secret-derived environment,
choose host or container mode, receive terminal callbacks, poll as a fallback,
cancel tasks, and expose live log metadata.

Evidence: [`pkg/components/runner/`](../../../pkg/components/runner/),
[`pkg/public/runner_live_log_session.go`](../../../pkg/public/runner_live_log_session.go),
and [`pkg/public/admin_runner_tasks.go`](../../../pkg/public/admin_runner_tasks.go).

## Core Components

```component
name: RunnerExecution
container: Workers
responsibilities:
  - Validating `runner.Spec` and creating authenticated broker tasks
  - Mapping terminal `runner.Task` state to passed/failed output channels
```

```component
name: RunnerCallbackAndLogs
container: API and Web
responsibilities:
  - Receiving broker webhooks and locating execution by `task_id`
  - Issuing/fetching live-log sessions and exposing active tasks to admins
```

#RunnerExecution is an action implementation resolved by #RegistryRuntime and
runs inside the workflow execution capability. It submits `CreateTaskParams` to the task broker,
stores `task_id` in execution KV and metadata, and schedules a poll hook.
#RunnerCallbackAndLogs handles broker callbacks; both poll and callback paths
converge on `processBrokerTaskStatus`, whose finished-state check makes duplicate
terminal signals harmless.

## System Contracts

### Key Contracts

- `TASK_BROKER_BASE_URL` and `TASK_BROKER_AUTH_TOKEN` are required to create a
  broker client.
- `runner.Spec` requires commands and machine type, validates environment
  sources, and requires an image for Docker mode.
- Execution timeout defaults to one hour and is bounded to 1–86,400 seconds.
- Broker task creation has a 30-second HTTP timeout and must return `201` with a
  task ID.
- `task_id` is persisted before the first poll. Callbacks locate an execution
  through that KV and return not found when no execution owns the task.
- Poll failures reschedule rather than finish the execution. Terminal status is
  successful only for `succeeded` with exit code zero.
- Cancellation treats broker `404` as already complete and briefly retries
  `409` while broker linkage settles.
- Secret-derived environment values are resolved through runtime secret
  contexts; the broker boundary receives plaintext required for execution.

### Integration Contracts

- Broker endpoints include `POST /v1/tasks`, task status/list, cancel, webhook,
  and live-log session operations.
- Webhook payloads follow `runner.Task`; log sink metadata is copied into
  execution metadata for the UI.
- The task broker and fleets are external boundaries, not SuperPlane
  containers.

## Architecture Decision Records

### ADR-001: Combine callback completion with polling fallback

**Context:** Broker callbacks minimize latency but can be delayed or lost.

**Decision:** Register a callback URL and also schedule status polls until a
terminal state is observed.

**Consequences:** Completion tolerates callback delivery failure. Duplicate
signals must remain idempotent and polling adds broker traffic.

## Open Questions

- What broker task retention guarantees underpin historical live-log access?
- Should fleet/machine availability be cached and validated before workflow
  commit?
