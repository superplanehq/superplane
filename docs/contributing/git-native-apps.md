# Git-Native Apps

SuperPlane stores each app's specification in a dedicated git repository. The repository is the source of truth for:

- `canvas.yaml` — workflow graph (nodes, edges, change management)
- `console.yaml` — console panels and layout
- Additional files under the repository root (scripts, docs, assets)

Database rows under `workflow_versions` are **materialized projections** of git commits, not an alternate editing surface.

## Branches

| Branch | Purpose |
|--------|---------|
| `main` | Live app; merged/published spec |
| `drafts/<user-id>` | Default per-user draft branch (CLI/agents) |
| `drafts/*` | Additional named draft branches |

Version IDs are **git commit SHAs** (40-character hex). Each commit on a branch produces an immutable `workflow_versions` row tagged with `git_branch` and `materialization_status`.

## Editing flows

### UI (IndexedDB staging)

The workflow builder stages local edits until the user clicks **Commit**, which calls `CommitCanvasRepositoryFiles` with the active draft branch and an expected head SHA. **Publish** merges the draft branch to `main` and runs live materialization.

### CLI

```bash
# Ensure a draft branch exists (idempotent for the default branch)
superplane apps drafts create [app-name-or-id]

# List draft branches with tip SHA and materialization status
superplane apps drafts list [app-name-or-id]

# Commit canvas.yaml to your draft branch
superplane apps canvas update --draft -f canvas.yaml

# Commit console.yaml (and canvas.yaml when already on the branch) atomically
superplane apps console set --draft -f console.yaml

# Commit arbitrary paths
superplane apps repository commit --path ./README.md --message "docs"

# Read materialized draft or live spec
superplane apps canvas get --draft -o yaml
superplane apps console get --draft -o yaml

# Publish when change management is disabled
superplane apps canvas update -f canvas.yaml   # commits then merges to main
```

### External git push

Clone/push against the app git remote (`/git/<canvas-id>.git`). Creating a new `drafts/*` branch via git (push a new ref) is the write path; SuperPlane derives draft metadata from git:

1. The public git proxy publishes `repository_branch_updated` events after a successful push.
2. The **RepositoryMaterializer** worker calls `SyncDraftBranchFromGit`, which registers a `canvas_draft_branches` row (auto-named `Draft #1`, `Draft #2`, …) and materializes the branch tip.

The UI and CLI `CreateDraftBranch` API only creates the git branch; the same sync path registers metadata and materializes the tip synchronously for API callers.

```bash
git push origin HEAD:refs/heads/drafts/$(superplane me -q id)
# draft appears in `superplane apps drafts list` after the worker processes the push
```

## APIs

| RPC | Role |
|-----|------|
| `CreateDraftBranch` | Create `drafts/*` git branch from `main`; sync registers metadata (`Draft #n`) |
| `ListDraftBranches` / `DeleteDraftBranch` | Manage draft metadata |
| `CommitCanvasRepositoryFiles` | Atomic multi-file commit + sync draft materialization |
| `PublishCanvas` | Merge draft → `main` + sync live materialization |
| `DescribeCanvasVersion` | Read materialized spec by SHA |

Removed RPCs (git-first only): `UpdateCanvasVersion`, `ApplyCanvasVersionChangeset`, `ValidateCanvasVersionChangeset`, `UpdateCanvasDashboard`, `CreateCanvasVersion`.

## Materialization

| Trigger | Timing |
|---------|--------|
| New `drafts/*` git branch (API, CLI, or push) | `SyncDraftBranchFromGit` registers metadata + materializes tip (sync for API; async worker for push) |
| `CommitCanvasRepositoryFiles` on a draft branch | Synchronous draft materialization |
| External git push updating an existing draft branch | Async worker + websocket `repository_branch_updated` |
| `PublishCanvas` | Synchronous live materialization + `CanvasPublisher` |

## Change management

When change management is enabled, operators commit to a draft branch and open a **change request** referencing the draft tip SHA. Publishing merges through the change-request flow instead of direct `PublishCanvas`.

See [Canvas Change Requests](canvas-change-requests.md).
