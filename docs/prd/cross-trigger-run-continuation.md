# Cross-Trigger Run Continuation

## Overview

This PRD defines a SuperPlane capability that allows multiple trigger events in the same workflow
to continue a shared run context instead of always starting isolated runs.

The core behavior is session-based: triggers can emit events with a continuation key, and the
workflow engine uses that key to resume the latest matching run session.

This enables workflows like PR automation where:

- A comment trigger starts a preview flow.
- A later PR closed trigger continues the same context for cleanup.
- Optional future events (for example, slash commands) can keep extending the same session.

## Problem Statement

Today, every trigger event starts a new root run. This prevents natural multi-event workflows from
carrying state across trigger boundaries.

As a result:

- Users duplicate logic to recover prior state on every trigger.
- Cleanup flows (for example, deleting preview sandboxes on PR close) are harder to implement.
- Workflows become brittle because correlation is manual and component-specific.
- Trigger-level orchestration does not match real-world event lifecycles.

The product needs a first-class way to correlate and resume context across trigger events.

## Goals

1. Allow trigger events to continue an existing workflow run context when a continuation key matches.
2. Preserve expression ergonomics so resumed runs can access prior context using existing mechanisms.
3. Keep current behavior as default (fresh run) when continuation is not configured.
4. Provide deterministic, auditable continuation behavior that is safe under concurrent events.
5. Support GitHub PR lifecycle workflows first, then generalize for other integrations.

## Non-Goals

- Replacing all explicit state storage patterns across components.
- Merging arbitrary concurrent branches into one linear timeline automatically.
- Changing existing component contracts for `Execute`, `Pass`, or payload shapes.
- Introducing cross-workflow continuation (scope is within one workflow/canvas).
- Delivering full visual timeline UX changes in v1.

## Primary Users

- **Workflow Builders**: Users wiring event-driven automations across resource lifecycles.
- **Platform Engineers**: Teams building reliable cleanup/closure automation.
- **Integration Authors**: Developers implementing trigger/component pairs for long-running processes.

## User Stories

1. As a user, I want a PR comment trigger to start a run and a PR closed trigger to continue it.
2. As a user, I want continuation to be optional and explicit per trigger flow.
3. As a user, I want resumed runs to access prior outputs without manual data plumbing.
4. As an integration author, I want a simple way to emit resumable trigger events using a key.
5. As an operator, I want clear logs/telemetry showing when continuation happened.

## Functional Requirements

### Session Model

- The engine must support a workflow-scoped run session identified by:
  - `workflow_id`
  - `continuation_key`
- A session stores:
  - Root event reference used by the session.
  - Latest execution reference used for continuation.
  - Timestamps for creation and last update.
- `workflow_id + continuation_key` must be unique.

### Trigger Event Continuation Metadata

- Trigger event emission must support optional continuation metadata:
  - `continuation_key` (string)
  - `continuation_mode` (at minimum: `start_or_resume`)
- If metadata is absent, behavior remains unchanged (fresh root run).

### Routing Behavior

- On root trigger events with continuation metadata:
  - If no session exists for key, create a session and process as a new root run.
  - If session exists, route as a resumed trigger event:
    - Keep session root event for run context lineage.
    - Use current trigger event as the incoming event for new queue items.
    - Set previous execution context from session latest execution when available.
- On execution completion/failure/pass in a continued session, update session latest execution.

### Expression and Context Semantics

- Existing expression primitives (`$`, `root()`, `previous()`) must continue to work.
- For resumed trigger events:
  - `root()` resolves from session root event.
  - `previous()` can resolve from session latest execution chain.
- If session latest execution is unavailable (for example, first event), expressions must still behave
  like a normal fresh run.

### Concurrency and Idempotency

- Continuation session read/update operations must be transaction-safe.
- Concurrent events for the same continuation key must not corrupt session pointers.
- Duplicate webhook deliveries should not create multiple sessions for the same key.

### Backward Compatibility

- Existing workflows and triggers must behave exactly as today unless continuation is configured.
- Existing integrations that emit trigger events without continuation metadata require no changes.

### GitHub Initial Productization

- Add continuation support to GitHub PR triggers as first adopter:
  - `github.onPRComment`
  - `github.onPullRequest` (including `closed`)
- Provide a default key strategy for PR workflows:
  - `<integration/repo identifier>:pr:<pr number>`
- Allow override for advanced users later, but default should require minimal setup.

## UX Requirements

- Workflow builder should expose continuation as an opt-in trigger setting.
- Defaults should avoid accidental continuation across unrelated events.
- Configuration copy should explain:
  - What key is used.
  - When a run is resumed versus started.
- Run history should indicate when a trigger event resumed an existing session.

## Data Model and API Scope (v1)

- Add a persistence model for workflow run sessions.
- Add internal event metadata support for continuation fields.
- Add router path for resumed trigger events.
- Add session update hooks when executions progress.
- Add trigger-level configuration support for GitHub PR triggers.

## Out of Scope (v1)

- Generic UI for custom continuation expressions across all integrations.
- Session expiration/TTL policies and archival UI.
- Manual session management APIs (pause/reset/rebind).
- Time-travel or replay tooling for session timelines.

## Acceptance Criteria

1. A workflow with `onPRComment` (start) and `onPullRequest closed` (resume) can share context via continuation key.
2. When continuation is enabled and a matching session exists, resumed trigger runs use prior session context.
3. When continuation is enabled and no session exists, workflow starts a new session automatically.
4. Workflows without continuation configuration preserve current behavior exactly.
5. Concurrent events with the same key do not create inconsistent session state.
6. Engine telemetry/logging can distinguish fresh runs from resumed runs.

## Success Metrics

- Reduction in workflow complexity for multi-trigger lifecycle automations.
- Increase in successful cleanup-on-close automations (for example PR sandbox teardown).
- Lower rate of user-added manual correlation components for common PR flows.
- No material regression in event routing latency for non-continuation workflows.

## Risks and Mitigations

- **Risk:** Continuation key collisions resume wrong sessions.  
  **Mitigation:** Scoped key design, safe defaults, clear docs, and per-trigger opt-in.

- **Risk:** Concurrency races cause pointer drift in latest execution tracking.  
  **Mitigation:** Transactional updates with row-level locking and deterministic update rules.

- **Risk:** Confusing mental model between fresh and resumed runs.  
  **Mitigation:** Explicit UX labels and run history indicators.

- **Risk:** Unexpected behavior in expressions using `previous()` on resumed flows.  
  **Mitigation:** Keep semantics consistent with existing chain resolution and add targeted tests.

## Rollout Plan

1. **Backend foundation**
   - Add session model/table.
   - Add continuation metadata plumbing.
   - Add resumed routing path.
2. **GitHub trigger adoption**
   - Add continuation configuration and key generation for PR triggers.
   - Add integration tests for PR comment -> PR close continuation.
3. **UI exposure**
   - Add trigger configuration controls and helper text.
   - Add run history labels for resumed events.
4. **Generalization**
   - Extend continuation options to more triggers/integrations.
   - Add advanced key customization where needed.

## Open Questions

1. Should v1 expose custom continuation key expressions in UI, or keep only safe defaults?
2. Should sessions have inactivity TTL by default, and if so, what duration?
3. If two triggers resume simultaneously, should we serialize processing per key or allow parallel continuation branches?
4. How should resumed-run lineage be represented in existing execution/history APIs for clients?
5. Do we need tenant-level guardrails limiting max active sessions per workflow?
