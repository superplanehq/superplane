# Deployment and Operations Feature Blueprint

## Feature Summary

Operators deploy SuperPlane as API/Web and worker processes backed by
PostgreSQL, RabbitMQ, and Git storage, configure public URLs/authentication and
secrets, probe health, scale stateless workloads, and optionally export
telemetry. These operational constraints are recorded in
[Technical Requirements](../../overview/technical-requirements.md); the current
feature requirements set does not define a deployment feature.

Evidence: [`release/superplane-helm-chart/helm/`](../../../release/superplane-helm-chart/helm/),
[`release/superplane-single-host-tarball/templates/docker-compose.yml`](../../../release/superplane-single-host-tarball/templates/docker-compose.yml),
[`Dockerfile`](../../../Dockerfile), and
[`pkg/server/server.go`](../../../pkg/server/server.go).

## Component Blueprint Composition

- [API and Web](../containers/api-and-web.container.md): public service,
  health/readiness, static UI, API, callbacks, and realtime.
- [Workers](../containers/workers.container.md): independently replicated
  asynchronous workload selected by `START_*` flags.
- [PostgreSQL](../containers/postgresql.container.md),
  [RabbitMQ](../containers/rabbitmq.container.md), and
  [Git Storage](../containers/git-storage.container.md): stateful dependencies
  supplied locally or externally.
- [Authentication and RBAC](../components/authentication-and-rbac.component.md):
  startup secrets and configured login/provider surfaces define the trust
  boundary.

## Feature-Specific Flow

The Helm chart creates separate API and worker deployments from the same image,
injects database/broker/auth/encryption/session/telemetry secrets, mounts OIDC
keys, and optionally creates local stateful services. Ingress/TLS routes port
`8000`. The single-host compose release runs a combined app process behind an
edge proxy and waits for healthy database, broker, and Git storage. Startup
validates mandatory configuration before serving.

## System Contracts

- `ENCRYPTION_KEY`, `BASE_URL`, `PUBLIC_API_BASE_PATH`, `JWT_SECRET`, and
  `OIDC_KEYS_PATH` are mandatory.
- API and workers use separate database pool sizing and may scale
  independently; stateful chart dependencies remain single replica by default.
- API Helm pods use a read-only root filesystem and drop Linux capabilities.
- Readiness/liveness probe HTTP availability, not end-to-end dependency health.
- Optional telemetry/error reporting/installation beacons are outbound
  boundaries and must not become correctness dependencies.
- WebSocket edge configuration must support upgrades and appropriate timeout.
- Backups must coordinate PostgreSQL and Git storage; RabbitMQ durability affects
  in-flight dispatch but is not canonical history.

## Requirement Coverage

- Feature-level requirement coverage is intentionally absent until a
  deployment requirements document defines user outcomes and acceptance
  criteria. Current technical coverage includes the container image,
  single-host packaging, Helm API/worker split, optional local stateful
  dependencies, ingress/TLS, probes, telemetry toggles, and scaling settings.

## Architecture Decision Records

### ADR-001: Package a modular monolith with role-selected processes

**Context:** API and asynchronous work share domain code but scale differently.

**Decision:** Ship one application image and use environment flags to run API,
  worker groups, or both.

**Consequences:** Releases stay synchronized and operational modes are flexible;
misconfigured flags can omit required workers or duplicate workloads.

## Open Questions

- Which production availability profiles should replace single-replica local
  PostgreSQL, RabbitMQ, and Git storage?
- Should startup/readiness verify required worker and dependency capability
  rather than only process HTTP health?
