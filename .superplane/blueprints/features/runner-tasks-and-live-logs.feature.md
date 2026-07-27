# Runner Tasks and Live Logs Feature Blueprint

## Feature Summary

Workflow authors configure command/script execution on a selected runner fleet,
choose host or container execution, inject environment and files, observe live
logs, receive structured results, cancel work, and let administrators inspect
active broker tasks. This feature implements the
[Runners](../../requirements/runners.md) requirements.

Evidence: [`pkg/components/runner/`](../../../pkg/components/runner/),
[`pkg/public/runner_live_log_session.go`](../../../pkg/public/runner_live_log_session.go),
[`pkg/public/admin_runner_tasks.go`](../../../pkg/public/admin_runner_tasks.go),
and [`web_src/src/ui/CanvasPage/RunnerLiveLogDialog/`](../../../web_src/src/ui/CanvasPage/RunnerLiveLogDialog/).

## Component Blueprint Composition

- [Runner Execution](../components/runner-execution.component.md):
  #RunnerExecution validates/submits/polls/cancels and
  #RunnerCallbackAndLogs handles callbacks and log access.
- [Workflow Execution](../components/workflow-execution.component.md):
  runner actions participate in normal execution state, output channels,
  retries, and cancellation.
- [Registry and Runtime](../components/registry-and-runtime.component.md):
  runner subcomponents and schemas are registered implementations.
- [Secrets and Runtime Configuration](secrets-and-runtime-configuration.feature.md):
  secret-backed environment is resolved only at execution.

## Feature-Specific Flow

The worker resolves the runner spec and creates an authenticated broker task
with fleet, mode, image, commands/script, environment, files, callback, and
timeout. It stores `task_id`, schedules polling, and waits. A broker callback or
poll updates log metadata and converges on terminal processing. The UI asks the
API for a live-log session derived from execution metadata; admins may list
non-terminal broker tasks.

## System Contracts

- Docker mode requires an image; host mode omits it.
- Machine type, commands, environment sources, and timeout are validated.
- Task IDs are the correlation key between broker state and node execution.
- Callback and poll races are safe because finished execution state is checked
  before emitting terminal output.
- Broker status `succeeded` is passed only with exit code zero; all other
  terminal outcomes use the failed channel.
- Cancellation is best effort against the broker and preserves local history.
- Broker authentication tokens and resolved environment secrets must not be
  exposed in UI/API payloads.

## Requirement Coverage

- **REQ-RNER-001:** Registered runner components and configuration schemas
  validate operation, fleet, mode, commands, environment, files, and timeout.
- **REQ-RNER-002:** Broker submission, callback/poll convergence, and workflow
  output channels expose successful results and terminal failures.
- **REQ-RNER-003:** Execution metadata creates authenticated live-log sessions
  used by the run inspector and live-log dialog.
- **REQ-RNER-004:** The administration endpoint lists non-terminal broker tasks
  behind installation-administrator authorization.

## Architecture Decision Records

### ADR-001: Delegate compute isolation to a task broker

**Context:** Arbitrary workflow commands should not execute inside API or worker
containers.

**Decision:** Submit authenticated tasks to an external broker/fleet and keep
only orchestration/correlation state in SuperPlane.

**Consequences:** Application containers stay isolated from user compute; broker
availability, security, and retention become explicit dependencies.

## Open Questions

- What task/log retention and redaction guarantees must brokers provide?
- How should capacity exhaustion or unavailable fleets appear in node setup
  versus execution?
