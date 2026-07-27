# API and Web Container Blueprint

## Container Summary

The API and Web container runs the `cmd/server` binary with
`START_PUBLIC_API=yes`. It combines #PublicHTTPServer, in-process
#GRPCGatewayServices, `ws.Hub`, authentication routes, OpenAPI assets, and the
compiled React/Vite application on port `8000`. It is the public trust boundary
for browser, CLI, webhook, and REST traffic.

Evidence: [`pkg/server/server.go`](../../../pkg/server/server.go),
[`pkg/public/server.go`](../../../pkg/public/server.go),
[`pkg/grpc/services.go`](../../../pkg/grpc/services.go),
[`protos/`](../../../protos/), and
[`web_src/`](../../../web_src/).

## Infrastructure

- The Helm API `Deployment` enables `START_PUBLIC_API`,
  `START_EVENT_DISTRIBUTER`, `START_WEB_SERVER`, and RBAC policy reload and
  exposes a `Service` on `8000`
  ([`api.yaml`](../../../release/superplane-helm-chart/helm/templates/api.yaml)).
- Every replica needs PostgreSQL, RabbitMQ, Git storage, encryption/JWT/session
  secrets, and mounted OIDC signing keys. The filesystem is read-only in Helm.
- The same application image can run workers, but Helm separates API and worker
  deployments so they scale independently.
- Ingress/TLS and optional telemetry exporters terminate outside this container.
  OpenTelemetry and error reporting are optional outbound boundaries.

## Entry Points and Boundaries

- `HTTP /api/v1/*` is matched from proto annotations and served by an
  in-process gRPC-Gateway `ServeMux`; no separate network gRPC hop exists.
- `/auth/*`, owner setup, health, webhook callbacks, runner callbacks,
  repository downloads, and administrative routes are native HTTP handlers.
- `/ws/{workflowId}` and the agent-session WebSocket endpoint attach authorized
  clients to the in-memory `ws.Hub`.
- Static routes serve the built frontend. The generated TypeScript client uses
  the OpenAPI representation of the proto contracts.
- Outbound calls go to PostgreSQL, RabbitMQ, Git Storage, configured identity
  providers, integration APIs, managed-agent providers, usage services, and a
  runner task broker.

Related components: [API Gateway and Realtime](../components/api-gateway-and-realtime.component.md),
[Authentication and RBAC](../components/authentication-and-rbac.component.md),
and [Git Staging and Versioning](../components/git-staging-and-versioning.component.md).

## System Contracts

### Key Contracts

- Unknown JSON request fields are rejected (`DiscardUnknown=false`), while
  response serialization emits unpopulated proto fields.
- Gateway authorization executes before business handlers and route failures
  pass through sanitized error handling.
- `BASE_URL`, `PUBLIC_API_BASE_PATH`, `JWT_SECRET`, `OIDC_KEYS_PATH`, and
  `ENCRYPTION_KEY` are startup requirements; missing values stop startup.
- WebSocket subscriptions are authenticated and scoped to a workflow or to the
  owning user’s `AgentSession`. The in-memory hub does not itself make events
  durable.
- The API persists canonical changes before publishing state notifications;
  clients must refetch after reconnect rather than treating realtime messages as
  the source of truth.

### Integration Boundaries

- #EventDistributer consumes RabbitMQ events and converts them into WebSocket
  payloads for this replica’s hub.
- #GRPCGatewayServices call model functions and shared components directly in
  process.
- Edge proxies must preserve WebSocket upgrades, forwarded identity/session
  headers expected by the server, and the configured base path.

## Architecture Decision Records

### ADR-001: Co-locate REST gateway, web UI, and realtime endpoints

**Context:** The public product surface shares authentication, authorization,
domain services, and deployment lifecycle.

**Decision:** Run gRPC-Gateway handlers, native HTTP routes, WebSockets, and
static web assets in one Go process.

**Consequences:** Deployment is simple and avoids an internal RPC hop. API and
WebSocket load scale together, and each replica needs an event distributor to
feed its process-local hub.

## Open Questions

- What sticky-session or fan-out guarantees are required for WebSockets when API
  replicas exceed one?
- Should native administrative and callback routes receive the same declarative
  authorization inventory used by proto-backed routes?
