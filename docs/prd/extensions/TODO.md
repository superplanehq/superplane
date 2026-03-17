# Next Steps

1. Implement the runtime execution plane.
   The current SDK includes worker WebSocket logic, but that transport should move out of the SDK and into a dedicated worker process. The first implementation should keep the execution plane fully in memory and split the work into the parts below.

   Phase 1: define the boundaries
   - Hub command: add a dedicated hub server under `cmd/extensions/hub/*`.
   - Hub packages: add orchestration, HTTP handlers, connection registry, and in-memory scheduling under `pkg/extensions/hub/*`.
   - Worker command: add a dedicated worker entrypoint at `cmd/extensions/worker/main.go`.
   - Worker packages: add registration, deregistration, job polling/streaming, bundle cache management, and Deno execution under `pkg/extensions/worker/*`.
   - SDK boundary: remove long-lived WebSocket and worker lifecycle behavior from `extensions/sdk/ts/*`; the SDK should only describe the extension, derive the manifest, and expose a CLI-like way of invoking operations on it.

   Phase 2: define the hub protocol
   - Registration endpoint: WebSocket upgrade at `/api/v1/register`, with `workerId` and `token` provided on the upgrade request.
   - The registration token is a SuperPlane-signed JWT that carries the worker claims. For now, that is organization ID and pool ID.
   - Successful registration upgrades the HTTP request to a WebSocket connection owned by the hub.
   - Deregistration should happen implicitly on WebSocket close, heartbeat timeout, or failed writes.
   - The protocol should support at least these hub-to-worker messages:
     - `job.assign`
     - `job.cancel`
     - `ping`
   - The protocol should support at least these worker-to-hub messages:
     - `job.complete`
     - `pong`
   - Each message should include stable IDs so the hub can correlate worker sessions, assigned jobs, bundle versions, and responses.
   - `job.assign` should include a bundle-access token distinct from the worker registration token.

   Phase 3: implement hub in-memory state and scheduling
   - Keep runtime state in memory for now:
     - connected workers by worker pool
     - worker session metadata
     - outstanding jobs by ID
     - assigned jobs by worker
     - bundle metadata cached by extension version
   - Scope workers to a worker pool, and worker pools to an organization.
   - Require the hub to validate that a worker can only register against its own organization/pool.
   - Add a small scheduler that:
     - finds an available worker in the target pool
     - marks the job as assigned before sending it
     - tracks in-flight jobs
     - times out or requeues work if the worker disconnects before completion
   - Start with a single-job-per-worker model. Concurrency per worker can be added later once Deno process isolation and cancellation rules are clearer.

   Phase 4: artifact delivery and bundle lookup
   - Reuse `pkg/extensions/storage.go` as the source of truth for extension artifacts.
   - Add public storage APIs to read `manifest.json` and `bundle.js` for a specific organization/extension/version; the hub should not depend on private helper methods.
   - Add a hub endpoint for workers to fetch a bundle by immutable version/digest reference.
   - The bundle fetch token should be distinct from the worker registration token and should carry organization ID, extension ID, and version ID claims.
   - Workers should download the bundle on first use, verify the digest, store it in a local cache directory, and reuse it on subsequent jobs.
   - Add cache invalidation rules for:
     - digest mismatch
     - local cache corruption

   Phase 5: implement the dedicated worker
   - The worker process owns all registration and transport logic. It should:
     - register with the hub
     - reconnect with the hub after connection loss
     - maintain the WebSocket session
     - accept jobs
     - fetch bundles on demand
     - execute jobs
     - return results, logs, and structured failures
   - The worker should keep a local cache keyed by extension version ID or digest.
   - Execution should happen in a Deno subprocess launched by the worker, not inside the hub and not inside the SDK transport layer.
   - The worker should create one execution per assigned job with explicit input/output handoff so failures are isolated to the job.
   - The first implementation can use simple local process execution and temp files/stdin-stdout pipes instead of a more advanced process pool.

   Phase 6: reshape the TypeScript SDK for worker-driven execution
   - Remove `startPackagedExtension()` as the primary runtime model for packaged bundles.
   - Replace the current SDK worker transport responsibilities with exports that a dedicated worker can import from the packaged bundle.
   - The bundle should expose at least:
     - manifest discovery
     - discovered operation metadata
     - an operation registry or invoke function that maps invocation targets to handlers
   - Keep invocation-envelope normalization and runtime-harness construction in the SDK, because those are runtime semantics of extension execution rather than transport concerns.
   - Update the packaging pipeline in `pkg/cli/commands/extensions/create_version.go` so runtime bundles target Deno-compatible ESM instead of Node/CommonJS worker bootstrap code.
   - Preserve a way to extract the manifest during packaging without depending on the long-lived worker transport entrypoint.

   Phase 7: integrate with the existing SuperPlane execution path
   - Do not add a public or separate internal API for submitting jobs to the hub.
   - The hub should be used by the existing core SuperPlane workers when they need to execute an extension-backed block.
   - In practice, this means the extension-aware execution path should be wired behind the same places that currently call block methods such as `Component.Execute()`.
   - `NodeExecutor` is the expected first integration point for component execution.
   - The execution path should:
     - resolve that the target block belongs to an extension
     - resolve the organization, worker pool, extension version, and invocation envelope
     - hand the execution request to the hub package directly in-process
     - wait for the correlated worker result
   - The hub remains an internal execution backend, not a new top-level product surface.
   - The hub should remain responsible for:
     - worker selection
     - timeout policy
     - result correlation
     - audit/log capture
   - The worker should remain responsible for:
     - execution of the extension operation
     - bundle lifecycle on disk
     - Deno process lifecycle

   Phase 8: testing
   - Hub package tests:
     - worker registration/unregistration
     - job assignment and reassignment on disconnect
     - pool scoping and org isolation
     - bundle fetch authorization
   - Worker package tests:
     - bundle download and cache reuse
     - digest verification
     - Deno execution success/failure paths
     - reconnect and deregistration behavior
   - SDK tests:
     - operation export shape
     - invocation dispatch through the exported registry
     - compatibility of manifest generation after transport removal
   - End-to-end tests:
     - worker registers to hub
     - first job downloads bundle and executes
     - second job reuses cached bundle
     - worker disconnect causes in-flight job failure or requeue as designed
     - `NodeExecutor` can execute an extension-backed component through the hub without exposing a separate submission API

   Initial implementation constraints
   - Keep hub state in memory.
   - Keep bundle storage backed by the existing extension artifact storage.
   - Avoid introducing a persistent job queue until the protocol and execution semantics are stable.
   - Prefer a narrow first slice: component/integration operation dispatch first, then cancellation/log streaming refinements.

2. Decide whether the watch mode should stay intentionally shallow or become recursive.
   The current CLI watches the entrypoint directory plus `integrations/`, `components/`, and `triggers/`.

3. Public registry of extensions.
   What we have right now is the equivalent of a private extension.
   However, it would be really good to have a public registry of extensions.
   Once we have that, we could even start focusing on implementing new integrations
   through extensions available in that registry instead of as part of SuperPlane itself.
