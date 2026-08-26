# SuperPlane Helm chart

## API health endpoints

The API exposes two unauthenticated, minimal health endpoints for cluster probes:

| Endpoint | Contract | Helm consumers |
| --- | --- | --- |
| `GET /health` | Process liveness. Returns `200` while the HTTP server can respond and does not check external dependencies. | Startup and liveness probes |
| `GET /ready` | Traffic readiness. Returns `200` when PostgreSQL responds within one second, otherwise a generic `503`. Concurrent checks are coalesced and the result is reused per API process for one second. | Readiness probe and GCE/ALB backend health checks |

Readiness responses use `Cache-Control: no-store` and never include database errors or connection details. Internal one-second result reuse bounds probe traffic to at most one PostgreSQL ping per API process per second; it can delay a readiness transition by at most that same interval.

The startup probe allows up to 22.5 minutes for the API to start before Kubernetes applies liveness and readiness probes. This covers the entrypoint's two separate migration-lock waits of up to ten minutes each, leaving 2.5 minutes of headroom for migrations and server startup. During a PostgreSQL outage, `/health` stays healthy so Kubernetes does not restart a live process, while `/ready` removes the pod from service traffic. The pod becomes ready again automatically after PostgreSQL recovers.

RabbitMQ, SuperGit, and external integrations are intentionally not part of API readiness so their outages do not remove partially functional API capacity. Requests that depend on an unavailable service can still fail and should be covered by dependency-specific monitoring; those dependencies are not pod-admission signals.
