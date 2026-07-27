# Workflow Triggers

## Overview

Triggers turn manual intent, schedules, webhooks, and integration events into
durable App runs. Workflow builders must be able to configure trigger inputs
and operators must be able to invoke eligible triggers with clear, observable
results.

## Terminology

- **Trigger:** A Canvas entry node that converts an incoming occurrence into a
  run.
- **Manual trigger:** A trigger an authorized person can invoke directly.
- **Trigger template:** A named input form for a manual invocation.

## Requirements

### REQ-TRIG-001: Configure event-driven triggers

**User story:** As a workflow builder, I want to configure webhook, schedule,
and integration-event triggers, so that Apps begin when relevant events occur.

**Acceptance criteria:**

- **AC-TRIG-001.1:** When an event matches an enabled trigger, SuperPlane shall
  start a run with the event payload available to the workflow.
- **AC-TRIG-001.2:** When an event does not match a trigger's configured
  conditions, SuperPlane shall not start a run for that trigger.

### REQ-TRIG-002: Invoke parameterized manual runs

**User story:** As an operator, I want to provide declared inputs when starting
an App manually, so that one workflow can safely handle multiple cases.

**Acceptance criteria:**

- **AC-TRIG-002.1:** When a manual trigger declares parameters, SuperPlane
  shall request those values and start the run with valid submitted inputs.
- **AC-TRIG-002.2:** When a required input is missing or invalid, SuperPlane
  shall not start a successful workflow and shall identify the input problem.

### REQ-TRIG-003: Manage webhook credentials

**User story:** As an App owner, I want to reset a webhook credential, so that
I can revoke an exposed endpoint secret.

**Acceptance criteria:**

- **AC-TRIG-003.1:** When an authorized owner resets a webhook credential,
  SuperPlane shall provide a replacement endpoint or secret.
- **AC-TRIG-003.2:** After reset, an invocation using the replaced credential
  shall not trigger the App.

### REQ-TRIG-004: Identify triggered runs

**User story:** As an operator, I want runs to carry a meaningful trigger-based
title, so that I can distinguish executions in operational history.

**Acceptance criteria:**

- **AC-TRIG-004.1:** When a trigger supplies a custom run title, SuperPlane
  shall display that title in run history.
- **AC-TRIG-004.2:** When no custom title is supplied, SuperPlane shall still
  identify the run by its trigger and status.

## Traceability

- **Product context:** [events and triggers](../../README.md#how-it-works)
- **API evidence:** [trigger catalog](../../protos/triggers.proto) and
  [trigger invocation and event operations](../../protos/canvases.proto)
- **Behavior evidence:** [parameterized manual runs](../../test/e2e/parameterized_manual_run_test.go),
  [webhook reset](../../test/e2e/webhook_reset_test.go), and
  [trigger run titles](../../test/e2e/trigger_run_title_test.go)
- **Feature blueprints:**
  [Workflow Runs and Observability](../blueprints/features/workflow-runs-and-observability.feature.md)
  and
  [Integrations and Event Ingestion](../blueprints/features/integrations-and-event-ingestion.feature.md)

## Open Questions

- What delivery, retry, and deduplication guarantees are promised per trigger
  class?
- Which trigger changes require republishing versus taking effect immediately?
