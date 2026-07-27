# SuperPlane Technical Blueprints

These blueprints describe the implemented SuperPlane modular-monolith
architecture. They are grounded in the Go server and workers, Protocol Buffer
contracts, React application, PostgreSQL schema, RabbitMQ messages, Git storage
provider, and release manifests. The generic templates in this directory remain
the starting point for future additions.

## Containers

- [API and Web](containers/api-and-web.container.md) — public HTTP boundary,
  in-process gRPC-Gateway services, WebSockets, and the built React application.
- [Workers](containers/workers.container.md) — independently scalable
  asynchronous orchestration and maintenance processes.
- [PostgreSQL](containers/postgresql.container.md) — authoritative transactional
  state.
- [RabbitMQ](containers/rabbitmq.container.md) — event notification and work
  dispatch.
- [Git Storage](containers/git-storage.container.md) — versioned repository
  content behind `git.Provider`.

Optional ingress/edge proxies and telemetry exporters are deployment boundaries,
not SuperPlane-owned application containers. Runner task brokers, managed-agent
providers, identity providers, email delivery, and integration APIs are external
service boundaries documented by their consuming components.

## Components

- [API Gateway and Realtime](components/api-gateway-and-realtime.component.md)
- [Authentication and RBAC](components/authentication-and-rbac.component.md)
- [Workflow Execution](components/workflow-execution.component.md)
- [Registry and Runtime](components/registry-and-runtime.component.md)
- [Integrations and Webhooks](components/integrations-and-webhooks.component.md)
- [Git Staging and Versioning](components/git-staging-and-versioning.component.md)
- [Managed Agents](components/managed-agents.component.md)
- [Runner Execution](components/runner-execution.component.md)

## Feature Compositions

- [Canvas Authoring and Versioning](features/canvas-authoring-and-versioning.feature.md)
- [Workflow Runs and Observability](features/workflow-runs-and-observability.feature.md)
- [Integrations and Event Ingestion](features/integrations-and-event-ingestion.feature.md)
- [Identity, Organizations, and Access](features/identity-organizations-and-access.feature.md)
- [Secrets and Runtime Configuration](features/secrets-and-runtime-configuration.feature.md)
- [Managed Canvas Agent](features/managed-canvas-agent.feature.md)
- [Runner Tasks and Live Logs](features/runner-tasks-and-live-logs.feature.md)
- [Console and Widgets](features/console-and-widgets.feature.md)
- [Canvas Memory](features/canvas-memory.feature.md)
- [Cross-App Orchestration](features/cross-app-orchestration.feature.md)
- [Deployment and Operations](features/deployment-and-operations.feature.md)

The [requirements index](../requirements/README.md) contains the
project-specific Feature Requirements Documents. Each feature blueprint links
to every applicable document and maps its actual `REQ-*` identifiers to a
technical path or an explicit implementation gap. Each requirements document
links back to the blueprint or blueprints that satisfy it, so traceability is
direct and bidirectional rather than inferred from index placement.

## Conventions

- `#ComponentName` names a runtime collaborator; backticked names identify
  concrete models, types, API contracts, routes, events, or state values.
- Links to source are evidence, not an exhaustive inventory.
- Dependencies state direction and the data crossing each boundary.
- ADRs record only decisions evident in source or deployment configuration.
- Failure semantics distinguish durable PostgreSQL state from best-effort
  RabbitMQ notification and external-provider behavior.
- Generic templates remain authoring contracts; files under `containers/`,
  `components/`, and `features/` contain project-specific designs.

## Cross-Cutting Invariants

- `Organization` is the tenant boundary for user-owned resources.
- PostgreSQL is authoritative; RabbitMQ accelerates dispatch and realtime
  delivery but workers retain polling/recovery paths where implemented.
- Only committed live workflow versions execute. Per-user staged files are not
  live until commit and publication succeed.
- Sensitive persisted values pass through configured encryption unless the
  explicitly unsafe `NO_ENCRYPTION=yes` development mode is selected.
- Generated API clients and OpenAPI artifacts derive from `protos/`.

## Cross-Cutting Open Questions

- Which RabbitMQ publishers require an outbox to close transaction/notification
  gaps?
- Which worker classes should receive separate deployments and scaling policies
  as load profiles diverge?
- What availability and backup targets should be standardized for PostgreSQL,
  RabbitMQ, and Git storage across supported deployment modes?
