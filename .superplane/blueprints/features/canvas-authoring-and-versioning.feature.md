# Canvas Authoring and Versioning Feature Blueprint

## Feature Summary

Users build a canvas graph, edit console and repository files, review a private
effective draft, and commit a named version that becomes the executable live
workflow. This feature implements the
[App Discovery and Lifecycle](../../requirements/app-discovery-and-lifecycle.md),
[Canvas Authoring](../../requirements/canvas-authoring.md), and
[Staging and Versioning](../../requirements/staging-and-versioning.md)
requirements.

Evidence: [`protos/canvases.proto`](../../../protos/canvases.proto),
[`pkg/grpc/actions/canvases/`](../../../pkg/grpc/actions/canvases/), and
[`web_src/src/pages/app/`](../../../web_src/src/pages/app/).

## Component Blueprint Composition

- [API Gateway and Realtime](../components/api-gateway-and-realtime.component.md):
  #GRPCGatewayServices expose canvas, repository, staging, version, folder, and
  preference contracts; `canvas-staging-updated` refreshes collaborators.
- [Git Staging and Versioning](../components/git-staging-and-versioning.component.md):
  #GitStagingVersioning owns autosaved `WorkflowStagedFile` operations and
  #CanvasPublisher promotes a commit.
- [Registry and Runtime](../components/registry-and-runtime.component.md):
  #RegistryRuntime validates implementation names and runs node `Setup`.
- [Authentication and RBAC](../components/authentication-and-rbac.component.md):
  `canvases:read/update/create/delete` governs the lifecycle.

## Feature-Specific Flow

The React canvas maintains local graph interactions and writes staged changes.
Repository reads resolve the user’s effective staged content. Commit supplies
author/message and expected live head, writes a Git revision, constructs the
draft `CanvasVersion`, applies the graph changeset, and promotes it to live.
Node deletion cancels affected active work; setup failures remain visible on
the committed node.

## System Contracts

- Autosave is not publication; only a successful commit changes live execution.
- Per-user staging must not leak between users.
- Stale staging cannot overwrite a newer live head.
- Node IDs remain unique, including against soft-deleted historical nodes.
- Edges whose source or target no longer exists are removed during publication.
- Realtime messages indicate changed state but clients recover by refetching.

## Requirement Coverage

- **REQ-APP-001:** Canvas and folder list services plus the home UI provide
  authorized App discovery and organization.
- **REQ-APP-002:** Canvas creation creates a blank App and routes its author to
  the editable App surface.
- **REQ-APP-003:** The public App-install preview and installation service
  validate repository definitions and bundled Console content before creation.
- **REQ-APP-004:** Canvas settings and deletion services update identity and
  remove deleted Apps from normal use.
- **REQ-AUTHOR-001:** Graph editing and staged workflow files add, configure,
  connect, arrange, and remove nodes.
- **REQ-AUTHOR-002:** Registry schemas, upstream-data autocomplete, and setup
  validation support expressions and reject invalid live graphs.
- **REQ-AUTHOR-003:** Repository-file APIs and App routes expose visual and file
  representations of the same effective draft.
- **REQ-AUTHOR-004:** `canvases:read` permits inspection while
  `canvases:update` guards staging and graph mutations.
- **REQ-STAGE-001:** Per-user `WorkflowStagedFile` state separates autosaved
  edits from the committed live version and supports discard.
- **REQ-STAGE-002:** Shared staged-state reads, realtime invalidation, and
  expected-head checks expose collaborator updates and stale edits.
- **REQ-STAGE-003:** Commit and publication create an immutable
  `CanvasVersion`, apply the graph changeset, and promote it to live.
- **REQ-STAGE-004:** Version APIs and historical preview render immutable prior
  versions independently of the live Canvas.

## Architecture Decision Records

### ADR-001: Execute immutable committed snapshots

**Context:** Concurrent editing must not change a run’s graph beneath it.

**Decision:** Keep editing in per-user staging and execute only the promoted
live `CanvasVersion`.

**Consequences:** Runs have a stable version reference; authors must explicitly
commit and resolve stale work.

## Open Questions

- Should stale stages support diff/merge instead of discard-only recovery?
- How should cross-store Git/PostgreSQL publication failures be surfaced and
  repaired?
