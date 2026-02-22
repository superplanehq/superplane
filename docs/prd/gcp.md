# GCP Integration

## Overview

The GCP integration connects SuperPlane to Google Cloud Platform, enabling workflows to manage GCP resources (Compute Engine VMs, etc.) and react to GCP events in real time. It supports two authentication methods: Service Account Key (JSON) and Workload Identity Federation (keyless OIDC).

This document captures the key design decisions made during the implementation.

## GCP API Client Strategy

### Approach: Google API Library for Types + Direct REST for API Calls

The GCP integration uses a hybrid approach:

- **`google.golang.org/api/compute/v1`** — used in the `CreateVM` component for typed Compute Engine structs (`compute.Instance`, `compute.AttachedDisk`, `compute.NetworkInterface`, etc.). These types are complex and deeply nested — hand-writing them would be error-prone and tedious to maintain.
- **`golang.org/x/oauth2/google`** — used for credential parsing and OAuth2 token management (Service Account Key and Workload Identity Federation).
- **Direct REST via `common.Client`** — used for Pub/Sub, Cloud Logging, IAM, and Service Usage API calls. These APIs require only simple CRUD operations (create/delete topic, create/delete sink, get/set IAM policy), where the request and response bodies are small and stable.

We do **not** use the higher-level Google Cloud Go SDK (`cloud.google.com/go/*`), which provides full gRPC-based service clients with automatic retry, pagination, and transport management.

### Why Not the Full Cloud Go SDK

**HTTP control.** SuperPlane routes all outbound HTTP through `core.HTTPContext` for observability and testing. The `common.Client` injects the Bearer token and delegates to `core.HTTPContext.Do()`. The Cloud Go SDK manages its own gRPC transport, making it harder to inject SuperPlane's HTTP client.

**Small API surface for eventing.** The event trigger infrastructure uses ~10 REST endpoints across Pub/Sub, Logging, IAM, and Service Usage. Each is a single function with a small JSON body. The overhead of pulling in `cloud.google.com/go/pubsub` and `cloud.google.com/go/logging` (with their gRPC stubs and transitive dependencies) is not justified for this surface area.

**Consistency.** The `common.Client` pattern (build URL → marshal JSON → `ExecRequest` → parse response) is used uniformly across all non-Compute API calls. Adding a second client abstraction (the Cloud SDK) would split the codebase into two different patterns for the same integration.

### Why We Use the API Library for Compute Types

The `google.golang.org/api/compute/v1` package provides auto-generated Go structs for Compute Engine resources. The `CreateVM` component builds complex `compute.Instance` payloads with nested disks, network interfaces, scheduling options, metadata, service accounts, and more. Using the library's types gives us compile-time safety and avoids maintaining ~50+ struct definitions by hand.

Note: we only use the **types** from this package. The actual API calls still go through `common.Client.ExecRequest` — we marshal the typed struct to JSON and send it via our own HTTP path.

### Comparison with AWS

The AWS integration uses the official `aws-sdk-go-v2` for all API calls. This is necessary because AWS request signing (SigV4) is complex and tightly coupled to the SDK's credential chain. GCP's REST APIs are simpler to call directly — authentication is a single `Authorization: Bearer {token}` header.

## Event Trigger Architecture: Eventarc vs. Cloud Logging Sink + Pub/Sub

### Options Considered

**Option A — Eventarc Advanced:** GCP's managed eventing service. Provisions a pipeline, enrollment (CEL filter), and message bus per trigger. Handles Pub/Sub topic management and CloudEvents formatting automatically.

**Option B — Cloud Logging Sink + Pub/Sub Push:** Custom pipeline using Cloud Logging sinks to filter audit logs and route them to a self-managed Pub/Sub topic with push delivery to SuperPlane.

### Decision: Option B (Cloud Logging Sink + Pub/Sub)

We initially implemented Eventarc Advanced but switched to Cloud Logging sinks for the following reasons:

**Region support.** Eventarc Advanced is only available in a subset of GCP regions. Cloud Logging sinks are project-level resources with no region restriction, so triggers work regardless of where the monitored resources are deployed.

**Resource simplicity.** Eventarc Advanced requires four resources per trigger (pipeline, enrollment, message bus, topic). Cloud Logging sinks require one resource per trigger (the sink), plus two shared resources per integration (topic + subscription). This reduces provisioning complexity and cleanup surface.

**Synchronous provisioning.** Eventarc resources use long-running operations that must be polled. Cloud Logging sink creation is synchronous — the API returns immediately, simplifying the action-based provisioning flow.

**Alignment with AWS pattern.** The AWS integration uses a shared SNS topic with per-trigger EventBridge rules, routed through `HandleRequest` to `OnIntegrationMessage`. Cloud Logging sinks map directly to this pattern: per-trigger sinks replace per-trigger EventBridge rules, shared Pub/Sub topic replaces shared SNS topic, and `HandleRequest` + `subscriptionApplies` handles routing in both.

**Cost.** Cloud Logging sinks and Admin Activity audit logs are free. Pub/Sub costs are based on message volume. Eventarc Advanced adds its own per-event charges on top of the underlying Pub/Sub costs.

### Trade-offs Accepted

- **No managed CloudEvents format.** Eventarc wraps events in CloudEvents automatically. With sinks, we parse raw audit log entries in `HandleRequest`. This is more code but gives full control over the event schema.
- **Manual IAM grants.** The sink's `writerIdentity` must be granted `roles/pubsub.publisher` on the topic. This requires `roles/pubsub.admin` on the integration's service account (see IAM section below).
- **No built-in retry/dead-lettering.** Eventarc provides some retry logic. With raw Pub/Sub push, unacknowledged messages are retried by Pub/Sub's own retry policy, but there is no dead-letter topic configured.

## Per-trigger Sinks vs. Shared Sink

### Options Considered

**Option A — Per-trigger sinks, shared topic:** Each trigger creates its own Cloud Logging sink with a narrow filter. All sinks route to the same Pub/Sub topic. `HandleRequest` uses pattern matching to dispatch events to the correct trigger.

**Option B — Single shared sink, application-level filtering:** One sink per integration captures all audit logs (broad filter). `HandleRequest` does all the filtering.

### Decision: Option A (Per-trigger Sinks)

**Reduced noise.** Per-trigger sinks only forward matching events, so the Pub/Sub topic only carries relevant messages. A shared sink would forward every audit log entry, increasing Pub/Sub costs and processing overhead.

**GCP-side filtering.** Cloud Logging's filter engine is optimized for high-volume log streams. Filtering at the source is more efficient than filtering in the application after delivery.

**Independent lifecycle.** Each sink is created and deleted with its trigger. Removing a trigger cleanly removes its sink without affecting other triggers on the same integration.

**Scalability.** GCP allows up to 200 sinks per project, which is sufficient for typical usage.

## Event Flow

```
GCP Audit Log → Cloud Logging Sink → Pub/Sub Topic → Push Subscription → SuperPlane HandleRequest
```

1. A GCP resource change generates an **Admin Activity audit log** entry.
2. The trigger's **Cloud Logging sink** matches the entry against its filter (e.g., `protoPayload.serviceName="compute.googleapis.com" AND protoPayload.methodName="v1.compute.instances.insert"`).
3. The sink forwards the matching log entry to the shared **Pub/Sub topic** (`sp-events-{integrationID}`).
4. The **push subscription** (`sp-sub-{integrationID}`) delivers the message as an HTTP POST to `{baseURL}/api/v1/integrations/{id}/events?token={secret}`.
5. `HandleRequest` verifies the token, decodes the Pub/Sub envelope (base64 → JSON log entry), and extracts an `AuditLogEvent`.
6. `subscriptionApplies()` matches the event against registered trigger subscriptions by `serviceName` and `methodName`.
7. Matching triggers receive the event via `OnIntegrationMessage` and emit it to the workflow.

### Authentication

The push subscription endpoint uses a shared secret token (generated per integration, stored as a SuperPlane secret). The token is passed as a query parameter and verified in `HandleRequest`. This avoids the complexity of OIDC JWT verification while keeping the endpoint protected.

## Required IAM Roles

| Role | Purpose |
|------|---------|
| `roles/logging.configWriter` | Create and delete Cloud Logging sinks |
| `roles/pubsub.admin` | Create topics/subscriptions and grant the sink's `writerIdentity` publisher access via `setIamPolicy` |
| Additional roles per component | e.g., `roles/compute.admin` for VM management |

`roles/pubsub.admin` is required because `EnsureTopicPublisher` calls `setIamPolicy` on the Pub/Sub topic to grant the sink's `writerIdentity` the `roles/pubsub.publisher` role. This permission is not included in `roles/pubsub.editor`. For security-conscious deployments, this could be replaced with a documented manual IAM grant step in the future.

## Adding a New GCP Event Trigger

To add a new trigger (e.g., Cloud Storage object creation):

1. Define a `SinkFilter` constant with the Cloud Logging filter for the target audit log events.
2. Implement `Setup()` — call `ctx.Integration.Subscribe(pattern)` and schedule the `provisionSink` action.
3. Implement `provisionSink()` — call `gcppubsub.CreateSink` with the filter and `gcppubsub.EnsureTopicPublisher`.
4. Implement `OnIntegrationMessage()` — validate and transform the `AuditLogEvent` before emitting.
5. Implement `Cleanup()` — call `gcppubsub.DeleteSink`.

The shared topic, subscription, and `HandleRequest` routing require no changes.
