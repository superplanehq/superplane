# Technical Requirements

## Security and Privacy

**Verified requirements:**

- Authenticate users via JWT cookies for the web UI, bearer tokens for API
  access, and optional OIDC.
- Enforce organization-scoped RBAC with Casbin before business logic executes;
  permissions are resource-action pairs.
- Store secrets encrypted; local secret providers keep key/value maps under
  organization domains.
- Support API keys with role assignment, optional expiration, and optional
  canvas scope.
- Keep AI assistance behind feature/org policy gates; canvas update permission
  is required for mutating agent flows and field suggestions.
- Redact or exclude sensitive configuration from AI prompts and assistant
  responses.
- Allow installation admins to control private-network egress policy, with
  environment variables able to override UI policy.

Evidence:
[docs/contributing/architecture.md:47-70](../../docs/contributing/architecture.md#L47-L70),
[protos/secrets.proto:122-150](../../protos/secrets.proto#L122-L150),
[protos/api_keys.proto:98-119](../../protos/api_keys.proto#L98-L119),
[docs/prd/inline-config-assistant.md:105-112](../../docs/prd/inline-config-assistant.md#L105-L112),
[README.md:184-196](../../README.md#L184-L196).

## Reliability and Recovery

**Verified requirements:**

- Persist events, queue items, runs, and executions so work survives process
  restarts.
- Process workflow steps asynchronously through RabbitMQ-backed workers.
- Support cancellation of executions and runs, re-emission of trigger events,
  and resolution of execution errors.
- Keep draft edits in per-user staging until an explicit commit promotes a new
  live version; require discarding stale staging before new commits.
- Support parent/child app composition with success, failure, and timeout
  routing.

Evidence:
[docs/contributing/architecture.md:29-45](../../docs/contributing/architecture.md#L29-L45),
[protos/canvases.proto:233-289](../../protos/canvases.proto#L233-L289),
[protos/canvases.proto:110-154](../../protos/canvases.proto#L110-L154),
[test/e2e/run_app_test.go:31-80](../../test/e2e/run_app_test.go#L31-L80).

**Unresolved:** formal availability SLOs, RPO/RTO, backup cadence, and
cross-region failover targets are not specified in the reviewed repository
docs.

## Performance and Scale

**Verified constraints and behaviors:**

- The architecture is a modular monolith whose API and workers can scale
  independently.
- Console panels are capped at 50 panels and 1 MiB panel payload size.
- Service-account design documents a quota of 100 service accounts per
  organization.
- Widget and list UIs paginate runs, versions, and related data; E2E tests cover
  large sidebar histories.

Evidence:
[docs/contributing/architecture.md:5-6](../../docs/contributing/architecture.md#L5-L6),
[docs/prd/console-and-widgets.md:1026-1028](../../docs/prd/console-and-widgets.md#L1026-L1028),
[docs/prd/service-accounts.md:247-252](../../docs/prd/service-accounts.md#L247-L252),
[test/e2e/runs_view_test.go:47-74](../../test/e2e/runs_view_test.go#L47-L74).

**Unresolved:** end-to-end latency budgets, sustained throughput targets, and
maximum concurrent runs per installation are not published as product
requirements in the reviewed sources.

## Compliance and Auditability

**Verified requirements:**

- Attribute canvas versions, cancellations, and related actions to users.
- Record approval decisions with approver configuration by user, role, or
  group.
- Keep run and execution history inspectable through APIs and UI.
- Prefer attributable machine identities over shared personal tokens for
  automation.

Evidence:
[protos/canvases.proto:607-619](../../protos/canvases.proto#L607-L619),
[test/e2e/approvals_test.go:30-38](../../test/e2e/approvals_test.go#L30-L38),
[docs/prd/service-accounts.md:7-20](../../docs/prd/service-accounts.md#L7-L20).

**Unresolved:** regulatory frameworks, retention periods for audit evidence,
and formal compliance certifications are not defined in the repository.

## Compatibility and Integrations

**Verified requirements:**

- Expose contracts through Protocol Buffers, gRPC, and gRPC-Gateway REST/OpenAPI
  endpoints; generate Go and TypeScript clients from those contracts.
- Provide registry-listed triggers and components across AI, VCS, CI/CD, cloud,
  observability, incident, communication, ticketing, and developer-tool
  categories.
- Support web UI, CLI, and API-key clients against the same authorization model.
- Keep frontend and backend Console YAML validation aligned; reject legacy
  `kind: Dashboard` on import.

Evidence:
[docs/contributing/architecture.md:9-12](../../docs/contributing/architecture.md#L9-L12),
[protos/canvases.proto:29-39](../../protos/canvases.proto#L29-L39),
[README.md:63-67](../../README.md#L63-L67),
[docs/prd/console-and-widgets.md:1019-1023](../../docs/prd/console-and-widgets.md#L1019-L1023).

## Deployment and Operations

**Verified requirements:**

- Support local demo, development Compose, single-host production, and
  Kubernetes deployments.
- Depend on PostgreSQL for state and RabbitMQ for messaging.
- Provide installation admin surfaces for settings and runner tasks.
- Prefer Docker-based development workflows; host-only Go/Node installs are not
  required for contributors.

Evidence:
[README.md:49-57](../../README.md#L49-L57),
[README.md:189-196](../../README.md#L189-L196),
[AGENTS.md:39-66](../../AGENTS.md#L39-L66),
[web_src/src/App.tsx:96-103](../../web_src/src/App.tsx#L96-L103).

## Engineering Constraints

**Verified repository constraints:**

- Backend language/runtime: Go, with pinned toolchain guidance in project docs.
- Frontend: TypeScript + React + Vite.
- Persistence/ORM: PostgreSQL via GORM; never hand-author migrations—use
  `make db.migration.create`.
- Do not hand-edit generated protobuf, OpenAPI, or SDK outputs; regenerate with
  `make pb.gen`.
- Prefer explicit `*gorm.DB` parameters in models; do not add new
  `database.Conn()` call sites in `pkg/models`.
- User-facing product name is **SuperPlane**.

Evidence:
[AGENTS.md:14-27](../../AGENTS.md#L14-L27),
[AGENTS.md:88-120](../../AGENTS.md#L88-L120),
[AGENTS.md:126-160](../../AGENTS.md#L126-L160),
[AGENTS.md:184-188](../../AGENTS.md#L184-L188).

## Open Questions

- What availability, latency, and throughput SLOs should self-hosted and cloud
  deployments commit to?
- What backup, restore, and disaster-recovery procedures are mandatory for
  production installs?
- What event, log, and agent-chat retention periods are required by default?
- Which compliance regimes, if any, are in scope for the next release cycle?
- What are hard multi-tenant fair-use limits beyond the documented console and
  service-account caps?
