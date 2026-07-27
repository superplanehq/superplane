# Personas

These personas are derived from implemented workflows, route permissions, and
product design documents. Their responsibilities are verified; demographic,
market-segment, and purchasing claims are not present in the repository.

## Primary Personas

### Workflow Builder

- **Role:** Engineer or platform practitioner who turns an engineering process
  into an app.
- **Goals:** Select triggers and actions, configure integrations and
  expressions, connect channels, validate changes, and publish a reviewable
  version.
- **Pain points:** Component discovery, required configuration, payload shape,
  and channel wiring require product and domain knowledge.
- **Key workflows:** Create an app; edit the canvas manually or with the agent;
  stage and commit changes; inspect a test run; configure memory and Console
  panels.
- **Successful outcome (inferred):** A valid, understandable workflow reaches a
  successful run quickly and can be changed without destabilizing the live
  version.

Evidence:
[docs/prd/ai-canvas-builder-sidebar.md:12-30](../../docs/prd/ai-canvas-builder-sidebar.md#L12-L30),
[test/e2e/canvas_staging_commit_publish_test.go:16-32](../../test/e2e/canvas_staging_commit_publish_test.go#L16-L32),
[test/e2e/agent_staging_edit_test.go:27-61](../../test/e2e/agent_staging_edit_test.go#L27-L61).

### Workflow Operator

- **Role:** Engineer, release manager, incident responder, or on-call operator
  responsible for live processes.
- **Goals:** Understand current state, launch approved work, diagnose failures,
  intervene safely, and recover or cancel work.
- **Pain points (inferred):** State is otherwise fragmented across tool-specific
  logs and dashboards; blind retries may duplicate or bypass guarded actions.
- **Key workflows:** Use app Consoles; inspect runs and node details; trigger
  parameterized runs; approve, push through, cancel, re-emit, or resolve work.
- **Successful outcome (inferred):** The operator can determine what happened
  and take an authorized corrective action without reconstructing the workflow
  from multiple systems.

Evidence:
[docs/prd/console-and-widgets.md:15-23](../../docs/prd/console-and-widgets.md#L15-L23),
[protos/canvases.proto:233-289](../../protos/canvases.proto#L233-L289),
[test/e2e/runs_view_test.go:21-45](../../test/e2e/runs_view_test.go#L21-L45).

## Secondary Personas

### Approver

A designated user, role holder, or group member who reviews a waiting step and
records approval. The runtime prevents the workflow from proceeding until its
requirements are satisfied.
([test/e2e/approvals_test.go:30-90](../../test/e2e/approvals_test.go#L30-L90))

### Organization Owner or Administrator

Sets up the installation or organization; manages members, groups, roles, API
keys, secrets, integrations, and AI feature policy. This persona owns access
boundaries rather than each app's implementation.
([web_src/src/App.tsx:96-127](../../web_src/src/App.tsx#L96-L127),
[docs/contributing/architecture.md:47-70](../../docs/contributing/architecture.md#L47-L70))

### Read-Only Stakeholder

Consumes app state, run history, and exported Console configuration without
editing or invoking work. End-to-end tests verify that viewers can read a
canvas but cannot enter edit mode or use ungranted agent features.
([test/e2e/canvas_permission_guards_test.go:17-48](../../test/e2e/canvas_permission_guards_test.go#L17-L48))

### Automation Client

A CI/CD pipeline, external agent, or script that accesses the API through an
API key. API keys support role assignment, expiration, and app scope, separating
machine access from a human browser session.
([protos/api_keys.proto:27-95](../../protos/api_keys.proto#L27-L95),
[protos/api_keys.proto:98-124](../../protos/api_keys.proto#L98-L124))

### SuperPlane Installation Operator

Deploys and maintains the application, PostgreSQL, RabbitMQ, runner capacity,
network policy, upgrades, and observability. Admin UI routes include
installation settings and runner tasks.
([README.md:189-196](../../README.md#L189-L196),
[web_src/src/App.tsx:96-103](../../web_src/src/App.tsx#L96-L103))

## Access and Responsibility Boundaries

| Boundary | Verified behavior |
| --- | --- |
| Tenant isolation | Resources are organization-scoped. |
| Canvas read vs update | Route gates and E2E permission tests separate viewing from editing and staging. |
| Runtime actions | Console runs and row actions require canvas update permission; the server remains authoritative. |
| Agent access | Agent UI is feature- and permission-gated. |
| Approvals | Approvers are constrained by user, role, or group configuration. |
| Machine identity | API keys carry explicit role and optional canvas scope. |

Evidence:
[docs/contributing/architecture.md:89-92](../../docs/contributing/architecture.md#L89-L92),
[docs/prd/console-and-widgets.md:1043-1053](../../docs/prd/console-and-widgets.md#L1043-L1053),
[test/e2e/canvas_permission_guards_test.go:17-48](../../test/e2e/canvas_permission_guards_test.go#L17-L48).

## Open Questions

- Which persona is the primary buyer versus the primary day-to-day user?
- How often do builders and operators overlap in one person or team?
- Which approval patterns dominate: individual users, roles, or groups?
- How should secondary personas such as auditors and compliance reviewers be
  formalized beyond current RBAC and run history?
