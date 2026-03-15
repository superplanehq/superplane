# Next Steps

This file captures the next implementation steps after the current manifest, SDK, packager, and GitHub example work.

## Current State

- Extension authoring uses `default defineExtension(...)`.
- The packager derives the serialized manifest and discovered operations.
- The engine and worker now exchange a stable invocation envelope.
- The Go entrypoints are now split into:
  - `cmd/cli` for user-machine workflows such as `package`, `register`, and `call`
  - `cmd/server` for SaaS/self-hosted runtime responsibilities
- The code layout is now split into:
  - `pkg/cli`
  - `pkg/server`
  - `pkg/protocol`
- `register` is now a CLI-side flow:
  - the CLI packages and inspects the extension locally
  - the CLI uploads the bundle, manifest, operations, and digest to the server
  - the server stores the registered extension without executing it for inspection
- `cmd/server` now uses `EXTENSIONS_DIR` as the extension storage root.
- The server now launches extension workers by running the packaged bundle directly with Node.
- `cli register --entry ...` packages into a temporary directory before upload, so the CLI does not need to keep registration artifacts around.
- Local server development now has:
  - a server `Dockerfile`
  - a `docker-compose.yml`
  - `make up` to build and start the server
- The reference example is a GitHub extension with:
  - `github` integration
  - `github.createIssue`
  - `github.closeIssue`
  - `github.onPush`

## Next Work

1. Implement server-side install from a registered extension.
   Registration and installation become separate concepts:
   - `register`: CLI uploads the extension package and metadata
   - `install`: server-side action that makes a registered extension available for a tenant/runtime

2. Implement real runtime-context population from the server.
   The current runtime harness is enough for packaging and dispatch validation, but the server still needs to populate the context contract intentionally:
   - `integration.getConfig`
   - `metadata`
   - `executionState`
   - `requests`
   - `events`
   - `webhook`

3. Add server-level dispatch tests.
   Add tests that prove the server invokes the correct handler and captures the expected runtime side effects for at least:
   - `integrations.github.sync`
   - `integrations.github.listResources`
   - `components.github.createIssue.execute`
   - `components.github.closeIssue.execute`
   - `integrations.github.webhook.setup`
   - `integrations.github.webhook.cleanup`

4. Improve the CLI/API shape.
   Move away from user-facing `--operation` strings like `components.github.createIssue.execute` and expose a cleaner invocation model based on:
   - block type
   - block name
   - operation

5. Start the sandbox-provider abstraction.
   Once the invocation contract is stable, define the provider boundary for:
   - artifact-backed extension startup
   - outbound control-channel lifecycle
   - Cloud Run as the first backend
   - future custom Kubernetes backends

## Out of Scope For The Immediate Next Slice

- Fine-grained permission declarations in the manifest
- Marketplace/install approval UX
- Multi-provider production support
- Full provider-specific integration brokering
- A permanent local bundle/install-state workflow in the CLI
