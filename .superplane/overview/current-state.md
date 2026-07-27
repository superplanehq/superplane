# Current State

## Verified Product State

SuperPlane is an open-source beta available as a self-hosted engine and as a
managed service. The core product models an app as a git-backed canvas,
optional console, app-scoped memory, and deterministic runtime. Canvases contain
trigger and action nodes connected by channel-bearing edges; incoming events
create runs whose executions and outputs are persisted.
([README.md:11-35](../../README.md#L11-L35),
[protos/components.proto:9-38](../../protos/components.proto#L9-L38),
[protos/canvases.proto:579-619](../../protos/canvases.proto#L579-L619))

The current user-facing route structure verifies these surfaces:

- organization home and app creation;
- app read, edit, settings, run inspection, version history, and console views;
- organization settings and installation administration.

Organization and app routes are protected by authentication and resource-action
permissions. Legacy canvas URLs redirect to app URLs.
([web_src/src/App.tsx:79-137](../../web_src/src/App.tsx#L79-L137))

## Existing Workflow in SuperPlane

1. A builder creates an app, adds triggers/actions, configures integrations and
   payload mappings, and connects nodes.
2. UI, CLI, and agent edits are staged per user. A commit with a message
   creates a version and promotes it to the live canvas; stale staging must be
   discarded.
3. A webhook, schedule, manual action, or component event starts a run. The
   router persists events and queue items, and workers execute eligible nodes
   asynchronously.
4. Operators inspect runs and node details, approve or push through waiting
   steps, cancel work, re-emit events, and resolve execution errors.
5. Teams can expose day-to-day status and controls through a per-app Console
   backed by runs, executions, and memory.

Evidence:
[docs/contributing/architecture.md:29-45](../../docs/contributing/architecture.md#L29-L45),
[docs/contributing/architecture.md:94-104](../../docs/contributing/architecture.md#L94-L104),
[test/e2e/canvas_staging_commit_publish_test.go:16-32](../../test/e2e/canvas_staging_commit_publish_test.go#L16-L32),
[docs/prd/console-and-widgets.md:9-23](../../docs/prd/console-and-widgets.md#L9-L23).

## Current Technical Shape

The system is a modular monolith: Go API and workers, gRPC contracts exposed
through a REST gateway, PostgreSQL persistence, RabbitMQ queues, and a
TypeScript/React/Vite frontend with WebSocket updates. Modules can be scaled
independently. ([docs/contributing/architecture.md:1-27](../../docs/contributing/architecture.md#L1-L27))

Authentication supports web sessions, bearer tokens, and optional OIDC.
Authorization is organization-scoped RBAC enforced in the gRPC interceptor.
API keys can be role- and app-scoped and can expire.
([docs/contributing/architecture.md:47-70](../../docs/contributing/architecture.md#L47-L70),
[protos/api_keys.proto:98-119](../../protos/api_keys.proto#L98-L119))

## Verified Strengths

- Durable, inspectable run/execution records with cancellation and version
  association.
- Human gates by user, role, or group, covered by end-to-end tests.
- Per-user staged editing that separates drafts from the live version.
- App composition, including parent apps that route child success, failure, and
  timeout outcomes.
- Broad integration categories and a registry-based component model.
- Consoles that combine live status, KPIs, runbooks, and authorized actions.

Evidence:
[protos/canvases.proto:704-743](../../protos/canvases.proto#L704-L743),
[test/e2e/approvals_test.go:19-92](../../test/e2e/approvals_test.go#L19-L92),
[test/e2e/run_app_test.go:31-80](../../test/e2e/run_app_test.go#L31-L80),
[README.md:63-67](../../README.md#L63-L67).

## Known Gaps and Risks

**Verified:** beta status permits breaking changes. AI-assisted surfaces are
feature- and permission-gated, and some design documents still describe
phased or v1 constraints. For example, field suggestions are one-shot and
string-only, while service-account token rotation is initially single-token.
([README.md:19](../../README.md#L19),
[docs/prd/inline-config-assistant.md:39-49](../../docs/prd/inline-config-assistant.md#L39-L49),
[docs/prd/service-accounts.md:171-180](../../docs/prd/service-accounts.md#L171-L180))

**Inferred:** breadth increases consistency and maintenance risk across
component schemas, backend validation, generated clients, UI mappers, and
integration APIs. The repository's synchronization checklists reinforce this
risk but do not quantify its user impact.

## Open Questions

- Which documented PRDs are fully shipped, partially shipped, or prospective?
- What production scale, availability, and recovery behavior has been observed?
- Which existing user workflows still depend on manual steps outside
  SuperPlane?
- What are the largest current sources of failed runs, abandoned canvases, or
  support requests?
