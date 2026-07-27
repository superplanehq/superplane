# Cross-App Orchestration

## Overview

Cross-App orchestration lets one App invoke a declared entry point in another
App and wait for its result. This enables reusable subflows while preserving
observable parent-child run relationships, parameter validation, outcome
routing, and timeout behavior.

## Terminology

- **Parent App:** The App whose workflow invokes another App.
- **Child App:** The invoked App.
- **On Run trigger:** A child entry point that declares accepted parameters.

## Requirements

### REQ-XAPP-001: Select an invokable App entry point

**User story:** As a workflow builder, I want to select another authorized App
and its declared entry point, so that I can reuse a workflow as a subflow.

**Acceptance criteria:**

- **AC-XAPP-001.1:** When configuring cross-App invocation, SuperPlane shall
  allow selection only among child Apps and entry points the builder may use.
- **AC-XAPP-001.2:** When the selected child App or entry point is unavailable,
  SuperPlane shall identify the broken dependency and prevent a successful
  invocation.

### REQ-XAPP-002: Pass validated parameters

**User story:** As a workflow builder, I want to map parent data into a child
App's declared parameters, so that the child starts with valid context.

**Acceptance criteria:**

- **AC-XAPP-002.1:** When mapped values satisfy the child entry point's
  declaration, SuperPlane shall start the child run with those values.
- **AC-XAPP-002.2:** When a required child parameter is missing or invalid,
  SuperPlane shall fail child initialization and route the parent through the
  failed outcome.

### REQ-XAPP-003: Route child outcomes to the parent

**User story:** As a workflow builder, I want the parent workflow to distinguish
child success and failure, so that it can continue with the appropriate
response.

**Acceptance criteria:**

- **AC-XAPP-003.1:** When the child run passes, SuperPlane shall complete the
  invocation through the parent's passed outcome.
- **AC-XAPP-003.2:** When the child run fails, SuperPlane shall expose the
  failure to the parent and continue only through its failed outcome.

### REQ-XAPP-004: Bound and trace child execution

**User story:** As an operator, I want parent and child runs linked and bounded
by timeout policy, so that nested automation is diagnosable and cannot wait
forever.

**Acceptance criteria:**

- **AC-XAPP-004.1:** When inspecting either run, SuperPlane shall retain enough
  relationship information to identify its parent or child counterpart.
- **AC-XAPP-004.2:** When the configured invocation timeout expires,
  SuperPlane shall cancel the child run and report a cancelled or failed child
  outcome to the parent.

## Traceability

- **Product context:** [multi-step App orchestration](../../README.md#what-it-does)
- **Runtime evidence:** [Run App component](../../pkg/components/messages/run_app.go)
- **Behavior evidence:** [cross-App success, failure, validation, and timeout](../../test/e2e/run_app_test.go)
- **Operational API evidence:** [runs and executions](../../protos/canvases.proto)
- **Feature blueprint:** [Cross-App Orchestration](../blueprints/features/cross-app-orchestration.feature.md)

## Open Questions

- Are cross-organization or cross-installation App invocations ever allowed?
- How are recursive invocation, maximum nesting, and child cancellation
  propagation constrained?
