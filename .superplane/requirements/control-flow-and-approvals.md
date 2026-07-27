# Control Flow and Approvals

## Overview

SuperPlane workflows need deterministic branching, fan-out, waiting, time
gates, and human decisions. These controls let builders encode guardrails while
giving operators enough context to act and understand which path continued.

## Terminology

- **Gate:** A node that delays or conditions downstream progress.
- **Approval:** A human decision that emits an approved or rejected outcome.
- **Channel:** The named outcome path emitted by a node.

## Requirements

### REQ-FLOW-001: Route by observable outcomes

**User story:** As a workflow builder, I want components to emit distinct
outcome channels, so that success, failure, and policy decisions follow the
right downstream path.

**Acceptance criteria:**

- **AC-FLOW-001.1:** When a component completes with a named outcome,
  SuperPlane shall continue only through connections subscribed to that
  outcome.
- **AC-FLOW-001.2:** When a branch condition does not match, SuperPlane shall
  not execute nodes reachable only through that branch.

### REQ-FLOW-002: Wait and resume durably

**User story:** As a workflow builder, I want a run to wait for a duration or
condition, so that long-running processes do not require external retry logic.

**Acceptance criteria:**

- **AC-FLOW-002.1:** When a run enters a configured wait, SuperPlane shall show
  it as incomplete and resume it when the wait condition is satisfied.
- **AC-FLOW-002.2:** When an operator cancels the waiting execution,
  SuperPlane shall stop that execution and show a cancelled outcome.

### REQ-FLOW-003: Enforce time gates

**User story:** As a release manager, I want work held until an allowed time,
so that operational changes occur within policy windows.

**Acceptance criteria:**

- **AC-FLOW-003.1:** When an item reaches a time gate outside its allowed
  window, SuperPlane shall hold it rather than execute downstream work.
- **AC-FLOW-003.2:** When the allowed time arrives, SuperPlane shall release
  the held item without requiring the workflow to be restarted.

### REQ-FLOW-004: Collect approval decisions

**User story:** As an approver, I want to approve or reject a pending workflow
request with visible context, so that guarded automation proceeds only with an
accountable decision.

**Acceptance criteria:**

- **AC-FLOW-004.1:** When an authorized approver submits a decision,
  SuperPlane shall record the actor and decision and route the workflow through
  the matching outcome.
- **AC-FLOW-004.2:** When a person is not an eligible approver, SuperPlane
  shall not accept their decision.

## Traceability

- **Product context:** [durable execution and approvals](../../README.md#what-it-does)
- **Catalog evidence:** [component definitions](../../protos/components.proto)
  and [action definitions](../../protos/actions.proto)
- **Behavior evidence:** [approvals](../../test/e2e/approvals_test.go),
  [wait behavior](../../test/e2e/wait_test.go), and
  [time gates](../../test/e2e/time_gate_test.go)
- **Design evidence:** [component outcome and operation guidance](../../docs/contributing/component-design.md)
- **Feature blueprint:** [Workflow Runs and Observability](../blueprints/features/workflow-runs-and-observability.feature.md)

## Open Questions

- Which quorum, delegation, expiration, and escalation policies are supported
  for approvals?
- What is the maximum supported wait duration and retention after resumption?
