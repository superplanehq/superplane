# SFDIPOT Product Factors Analysis: SuperPlane

## Heuristic Test Strategy Model (HTSM) -- James Bach's Framework

**Project**: SuperPlane -- Event-Driven Workflow Orchestration Platform
**Analysis Date**: 2026-03-28
**Overall Risk Assessment**: MEDIUM-HIGH

---

## Executive Summary

SuperPlane is a substantial event-driven workflow orchestration platform with 36 Go packages, 43 third-party integrations, a React 19 frontend, and a Python AI agent service. The system's reliance on polling-based workers with semaphore-bounded concurrency and RabbitMQ message passing creates a complex failure surface. 63 test ideas generated across all 7 factors, 14 exploratory test session proposals, and 14 clarifying questions.

| Priority | Count | Description |
|----------|-------|-------------|
| P0 (Critical) | 8 | Race conditions, security bypasses, infinite loops, data loss |
| P1 (High) | 19 | Auth gaps, missing timeouts, scalability limits |
| P2 (Medium) | 24 | Config risks, test coverage gaps, observability |
| P3 (Low) | 12 | Browser compat, documentation, non-critical UX |

---

## 1. STRUCTURE -- What the Product IS

### Strengths
- Clean registry pattern using Go `init()` for self-registering components/integrations
- Error sanitizer interceptor prevents internal error leakage
- Panic recovery middleware with Sentry integration
- Well-structured model layer with explicit table name mappings

### Key Risks
- **P0: Global database singleton** — `database.Conn()` called directly from models, tightly coupling to single connection pool
- **P1: Domain/DB naming mismatch** — Code uses "Canvas" but tables are still "workflows"
- **P1: 214 database migrations** in ~12 months — high velocity increases conflict/corruption risk
- **P1: Dual JWT library versions** — `golang-jwt/jwt/v4` and `v5` both in `go.mod`
- **P1: RabbitMQ 3.8.17 is EOL** — known issues with ordering and cluster partitions
- **P1: `HEALTHCHECK NONE` in production Docker image** — no monitoring outside Kubernetes
- **P2: Hardcoded encryption key in docker-compose.dev.yml** — risk if leaked to production

### Top Test Ideas
1. Build with `-race` flag and run full test suite to surface data races (P0)
2. Execute all 214 migrations up/down in sequence to confirm reversibility (P1)
3. Start application with RabbitMQ unavailable — does it panic, retry, or degrade? (P0)
4. Generate JWT with v4 claims, validate with v5 parsing to expose cross-version issues (P1)

---

## 2. FUNCTION -- What the Product DOES

### Core Capabilities
- Canvas (Workflow) management with versioning, change requests, conflict resolution
- Event-driven execution: Event -> EventRouter -> NodeQueueWorker -> NodeExecutor
- 15 components (approval, filter, if, merge, http, ssh, memory, etc.)
- 3 triggers (schedule, start, webhook)
- 43 integrations (AWS, GCP, GitHub, Slack, PagerDuty, etc.)
- Pessimistic locking via `SELECT FOR UPDATE SKIP LOCKED`

### Key Risks
- **P0: `GenerateUniqueNodeID` infinite loop potential** — loops forever if all IDs exhausted
- **P0: Polling fallback with minute-level latency** — if RabbitMQ fails silently, workflows delayed
- **P1: Semaphore limits hardcoded to 25** — no dynamic scaling
- **P1: Approval component has no timeout** — workflows can hang indefinitely
- **P1: `panic(err)` on missing RabbitMQ URL** — crashes entire server
- **P1: EventRouter silently continues on error** — potential infinite retry of poison events

### Top Test Ideas
5. Create canvas with 10,000 nodes, test `GenerateUniqueNodeID` collision boundary (P0)
6. Stop RabbitMQ with 50 queued events, restart, measure fallback processing time (P0)
7. Submit 100 concurrent events to same node — confirm exactly 100 executions, no duplicates (P0)
8. Create circular node references (A->B->C->A) — confirm linter catches it, engine doesn't loop (P0)
9. Inject malformed event payload, confirm EventRouter marks as failed vs infinite retry (P0)

---

## 3. DATA -- What it PROCESSES

### Core Data Flow
Event -> CanvasEvent (pending) -> EventRouter (routed) -> CanvasNodeExecution (pending -> started -> finished) -> Output Events -> Next Node

### Strengths
- Execution chain tracking via `RootEventID`, `PreviousExecutionID`, `ParentExecutionID`
- Soft deletes with 30-day grace period
- Event retention worker with configurable per-org windows
- Configuration snapshots on execution creation

### Key Risks
- **P0: 64KB event payload limit enforced only at API layer** — internal events may bypass
- **P1: Unbounded `ListPendingNodeExecutions`/`ListPendingCanvasEvents`** — OOM risk after outage
- **P1: Canvas memory has no size limit** — workflows can grow unbounded
- **P0: WebhookProvisioner 3-phase process** — crash between phases 2-3 creates duplicate webhooks
- **P1: No optimistic concurrency control on non-versioned canvas updates**

### Top Test Ideas
10. Create event with exactly 64KB payload (success), then 64KB+1 (reject) (P0)
11. Insert 10,000 pending events, start EventRouter, measure memory consumption (P1)
12. Simulate crash after webhook provisioning but before DB update — check for duplicates (P0)
13. Two users update same non-versioned canvas simultaneously — confirm consistency (P1)

---

## 4. INTERFACES -- How it CONNECTS

### APIs
- gRPC Internal (port 50051): 13 services
- REST Public (port 8000): gRPC-gateway with Swagger UI
- WebSocket: Real-time canvas updates
- Webhook endpoints: 43 integration receivers

### Key Risks
- **P1: WebSocket hub `Run()` has no exit condition** — prevents clean shutdown
- **P1: No rate limiting on inbound webhook endpoints** — flood risk
- **P0: Only 7 frontend spec files for ~138K lines TypeScript** — extreme coverage gap
- **P2: gRPC reflection enabled in production** — aids API enumeration

### Top Test Ideas
14. Open 500 WebSocket connections, broadcast event, measure delivery latency (P1)
15. Send 1,000 webhooks/sec to single endpoint — does it queue, drop, or 429? (P1)
16. Open canvas in two browser tabs, edit both, confirm WebSocket sync without data loss (P1)

---

## 5. PLATFORM -- What it DEPENDS ON

### Infrastructure
- PostgreSQL 17.5, RabbitMQ 3.8.17, Ubuntu 22.04, Go 1.25.3, Node.js
- OpenTelemetry, Sentry, Chromium (E2E tests)

### Key Risks
- **P1: Go 1.25 is bleeding edge** — potential compiler/stdlib instability
- **P1: RabbitMQ 3.8.17 is 5+ years old** with known CVEs
- **P2: No distributed cache** — in-process caching breaks multi-instance deployments
- **P2: WebSocket URL construction** may fail behind SSL-terminating proxy

### Top Test Ideas
17. Deploy two instances behind load balancer — confirm usage limit cache consistency (P1)
18. Kill RabbitMQ during active execution — confirm graceful degradation without data loss (P0)
19. Access app through nginx SSL reverse proxy — confirm WebSocket upgrades to `wss:` (P2)

---

## 6. OPERATIONS -- How it's USED

### Startup Sequence
1. Validate 6 required env vars → 2. Create database → 3. Run migrations → 4. Start server

### Key Risks
- **P0: `docker-entrypoint.sh` line 6** checks `$DB_PASSWORD` but prints "DB username not set" — wrong error message
- **P1: No migration version locking** — concurrent container starts cause migration conflicts
- **P1: 60+ environment variables** with no validation schema
- **P1: No HTTP health check endpoint** — gRPC has one but public API doesn't
- **P1: Installation admin promotion has no audit trail**

### Top Test Ideas
20. Start with `DB_PASSWORD` unset — confirm error correctly identifies the variable (P0)
21. Start two containers simultaneously — confirm migrations don't conflict (P1)
22. Promote user to admin, perform actions, demote — confirm immediate revocation (P1)
23. Delete organization, confirm soft-delete cascade, test recovery within 30 days (P1)

---

## 7. TIME -- WHEN Things Happen

### Concurrency Mechanisms
- Semaphore-bounded goroutines (25 per worker)
- `SELECT FOR UPDATE SKIP LOCKED` for execution locking
- `sync.RWMutex` in registry and WebSocket hub
- RabbitMQ consumer concurrency

### Key Risks
- **P0: WebSocket hub race condition** — `BroadcastToWorkflow` during client disconnect could panic
- **P0: NodeQueueWorker 1-second polling** creates unnecessary DB pressure under load
- **P1: No execution timeout** — hung HTTP calls leave executions in "started" forever
- **P1: Event retention worker may corrupt execution chains** if deleting during active execution
- **P1: No circuit breaker on integration calls** — down APIs exhaust semaphore slots
- **P1: RabbitMQ doesn't guarantee message ordering** — may cause incorrect execution sequences

### Top Test Ideas
24. 5 workers processing same canvas events — confirm no double-execution or lost events (P0)
25. During broadcast to 100 WS clients, disconnect 50 simultaneously — confirm no panics (P0)
26. Configure HTTP component to never-responding server — confirm execution eventually times out (P0)
27. Trigger event retention on root event while child is still running — confirm safe handling (P1)
28. Send 100 sequential events rapidly — confirm processing preserves order (P1)

---

## Exploratory Test Session Proposals

| # | Session | Factor | Priority |
|---|---------|--------|----------|
| 1 | **Dependency Surgery** — Remove integration import, explore system behavior | Structure | P2 |
| 2 | **Migration Archaeology** — Verify 5 random migration up/down reversibility | Structure | P1 |
| 3 | **Canvas Complexity** — Build increasingly complex canvases until engine breaks | Function | P0 |
| 4 | **Integration Failure Modes** — Invalid credentials for top 5 integrations | Function | P1 |
| 5 | **Expression Injection** — Adversarial inputs via `{{...}}` template syntax | Function | P0 |
| 6 | **Data Boundary Walking** — Push every field to extremes | Data | P1 |
| 7 | **Secret Lifecycle** — Create/update/use/delete/re-create secrets | Data | P1 |
| 8 | **WebSocket Resilience** — Throttle to 3G, trigger rapid events | Interfaces | P1 |
| 9 | **API Enumeration** — Use gRPC reflection to enumerate and probe all methods | Interfaces | P2 |
| 10 | **Resource Starvation** — 256MB memory, pool size 1 | Platform | P1 |
| 11 | **Chaos Configuration** — Randomized env var combinations | Operations | P1 |
| 12 | **Upgrade Simulation** — Run old schema, insert data, migrate to current | Operations | P1 |
| 13 | **Thundering Herd** — 100 webhooks simultaneously | Time | P0 |
| 14 | **Long-Running Workflow** — 7-day wait component across restarts | Time | P1 |

---

## Clarifying Questions

1. Is there an ADR for the Canvas/Workflow naming migration?
2. What is the intended deployment topology (single vs multi-instance)?
3. What happens when a node execution hangs indefinitely? Is there a planned timeout?
4. Is the canvas linter enforced before publication or advisory only?
5. What is the recovery path for failed webhook provisioning (idempotency)?
6. What are the data retention defaults and limits per organization?
7. Is there a maximum canvas memory size per workflow?
8. Are WebSocket connections authorized at the organization level?
9. Is the OpenAPI spec autogenerated or manually maintained?
10. Why does `go.mod` specify Go 1.25? (not in standard release schedule)
11. Is there a plan to upgrade RabbitMQ from 3.8.17?
12. What is the backup and disaster recovery strategy?
13. How does the system handle clock skew between instances and database?
14. What is the expected maximum event throughput per canvas per second?

---

## Automation Fitness

| Type | Count | % | Rationale |
|------|-------|---|-----------|
| Unit | 19 | 30% | Config validation, error sanitizer, expression parsing, JWT |
| Integration | 26 | 41% | Worker behavior, DB locking, migration testing, API contracts |
| E2E | 10 | 16% | Canvas editor flows, approval flows, WebSocket real-time |
| Human Exploration | 8 | 13% | Complex failure modes, UX quality, adversarial discovery |

---
*Generated by AQE v3 Product Factors Assessor Agent (HTSM/SFDIPOT)*
