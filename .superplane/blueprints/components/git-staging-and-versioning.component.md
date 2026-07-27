# Git Staging and Versioning Component Blueprint

## Capability Summary

Git staging and versioning separates per-user canvas edits from the committed
live workflow. It stages canvas, console, and repository file operations,
detects stale bases, commits a repository revision, materializes a
`CanvasVersion`, runs node setup, and promotes the resulting graph to live.

Evidence: [`protos/canvases.proto`](../../../protos/canvases.proto),
[`pkg/grpc/actions/canvases/`](../../../pkg/grpc/actions/canvases/),
[`pkg/grpc/actions/canvases/changesets/`](../../../pkg/grpc/actions/canvases/changesets/),
[`pkg/models/canvas_version.go`](../../../pkg/models/canvas_version.go), and
[`pkg/git/provider/provider.go`](../../../pkg/git/provider/provider.go).

## Core Components

```component
name: GitStagingVersioning
container: API and Web
responsibilities:
  - Managing per-user `WorkflowStagedFile` operations
  - Committing files with expected-head concurrency and promoting `CanvasVersion`
```

```component
name: CanvasPublisher
container: API and Web
responsibilities:
  - Diffing live and proposed node/edge graphs
  - Applying setup, deletion, cancellation, and live promotion transactionally
```

#GitStagingVersioning depends on Git Storage for branches/commits and PostgreSQL
for stage/version metadata. It calls #CanvasPublisher with #RegistryRuntime,
encryption, auth, webhook, and repository capabilities. #CanvasPublisher inserts
all new node rows before deferred `Setup` so self/sibling references can resolve,
filters edges to existing nodes, and records execution/queue cleanup caused by
deletion.

## System Contracts

### Key Contracts

- Staging belongs to a user and workflow; reads used by editing tools expose the
  effective staged content for that user.
- A stage based on an old live head is stale and must be discarded before a new
  commit.
- `ExpectedHeadSHA` prevents committing over a branch that moved concurrently.
- Commit is the boundary that makes edits live; autosave alone never changes
  the executable graph.
- A node’s implementation type/name cannot be changed in place; delete and add
  is required.
- Widget nodes live in version content but are not persisted as executable
  `workflow_nodes`.
- `Setup` errors persist the node in an error state rather than silently
  treating setup as success.
- Git and PostgreSQL cannot share a transaction. Failures must be surfaced and
  reconciled rather than claiming global atomicity.

### Integration Contracts

- REST operations include `PUT/GET/DELETE /staging`,
  `POST /staging/commit`, repository file listing, and version listing/detail.
- Agent `patch_staging` and the web/CLI stage paths share the same model.
- See [Git Storage](../containers/git-storage.container.md) and
  [Registry and Runtime](registry-and-runtime.component.md).

## Architecture Decision Records

### ADR-001: Require stage then commit for all live edits

**Context:** Workflow configuration and related files need reviewable history,
optimistic concurrency, and a stable execution snapshot.

**Decision:** Store user edits in staging and promote only a committed version
to the live graph.

**Consequences:** Execution sees a stable version and changes have authorship
and messages. Users must resolve stale staging and cross-store failures require
careful recovery.

## Open Questions

- What automated reconciliation detects a Git commit that succeeded before
  PostgreSQL promotion failed?
- Should users be able to inspect or merge stale staging rather than only
  discard it?
