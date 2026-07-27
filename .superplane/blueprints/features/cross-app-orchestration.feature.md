# Cross-App Orchestration Feature Blueprint

## Feature Summary

Workflow authors invoke an authorized App's declared On Run entry point, pass
validated parameters, wait for completion, and route the child result back to
the parent with an inspectable run relationship. This feature implements the
[Cross-App Orchestration](../../requirements/cross-app-orchestration.md)
requirements.

Evidence: [`pkg/components/messages/run_app.go`](../../../pkg/components/messages/run_app.go),
[`pkg/workers/contexts/run_execution_context.go`](../../../pkg/workers/contexts/run_execution_context.go),
[`pkg/workers/run_callback_dispatcher.go`](../../../pkg/workers/run_callback_dispatcher.go),
and [`test/e2e/run_app_test.go`](../../../test/e2e/run_app_test.go).

## Component Blueprint Composition

- [Registry and Runtime](../components/registry-and-runtime.component.md):
  the registered `runApp` action validates the selected App, On Run trigger,
  parameter configuration, and timeout.
- [Workflow Execution](../components/workflow-execution.component.md):
  run contexts create linked child runs, callback dispatch resumes the waiting
  parent execution, and normal cancellation/finalization handles both runs.
- [Authentication and RBAC](../components/authentication-and-rbac.component.md):
  App and node lookup remains organization- and permission-scoped during setup
  and invocation.
- [API Gateway and Realtime](../components/api-gateway-and-realtime.component.md):
  run descriptions serialize parent and child references for inspection.

## Feature-Specific Flow

During setup, `runApp` resolves the configured App and On Run trigger and stores
stable metadata. Execution calls the run context with mapped parameters and
callbacks for child entry and completion, then schedules a timeout and waits.
The child run stores parent workflow, run, and execution references. Completion
dispatch invokes the parent hook, which emits `passed` only for a passed child
and `failed` for every other terminal result. Timeout or parent cancellation
cancels the child through the run context.

## System Contracts

- The target node must be an On Run trigger selected from an App available to
  the current organization and authorization context.
- Required On Run parameters are validated before a successful child start.
- A parent execution waits for exactly one linked child run terminal result.
- Child runs retain parent workflow, run, and execution identifiers; parent run
  descriptions group child references by invoking execution.
- Callback handling checks terminal execution state so duplicate completion and
  timeout delivery cannot emit a second outcome.
- Timeout defaults to 3,600 seconds and may be configured to a positive value.
- Child pass maps to `passed`; failure, cancellation, and timeout map to
  `failed` with child result metadata.

## Requirement Coverage

- **REQ-XAPP-001:** App and App-node configuration fields restrict selection to
  resolvable Apps and On Run trigger nodes, and setup fails broken references.
- **REQ-XAPP-002:** Run-parameter configuration derives from the selected
  trigger and child creation validates required values before initialization.
- **REQ-XAPP-003:** The completion callback maps the child's terminal result to
  the parent's `passed` or `failed` output channel with run metadata.
- **REQ-XAPP-004:** Parent identifiers on `CanvasRun`, serialized run
  references, scheduled timeout handling, and child cancellation provide
  bounded, bidirectionally inspectable execution.

## Architecture Decision Records

### ADR-001: Model App invocation as a linked child run

**Context:** Reusable subflows need normal run durability and observability
without making the parent execute another App's graph directly.

**Decision:** Create a normal child `CanvasRun` carrying parent references and
resume the waiting parent through a completion callback.

**Consequences:** Child execution uses existing queues, history, and
cancellation; callback idempotency and nested-run limits remain important.

## Open Questions

- What recursion and maximum nesting limits should be enforced?
- Should parent cancellation wait for confirmed child cancellation before the
  parent execution becomes terminal?
