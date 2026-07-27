# Git Storage Container Blueprint

## Container Summary

Git Storage persists versioned workflow repository files behind the
`git.Provider` interface. The standard release uses the `supergit` service over
HTTP; code also contains alternative and in-memory providers. Repository
content complements, rather than replaces, PostgreSQL workflow metadata and
runtime state.

Evidence: [`pkg/git/provider/provider.go`](../../../pkg/git/provider/provider.go),
[`pkg/git/supergit/`](../../../pkg/git/supergit/),
[`pkg/git/codestorage/`](../../../pkg/git/codestorage/), and
[`supergit.yaml`](../../../release/superplane-helm-chart/helm/templates/supergit.yaml).

## Infrastructure

- The Helm chart runs a single-replica `StatefulSet` with a `ReadWriteOnce`
  volume at `/var/lib/supergit/repos`, HTTP port `8080`, and `/health` probes.
- The provider is selected with `GIT_STORAGE_PROVIDER`; the chart configures
  `supergit` and `GIT_STORAGE_SUPERGIT_BASE_URL`.
- Default branch, maximum file size, and maximum commit size are Git-storage
  service configuration. Chart defaults are `main`, 10 MiB per file, and
  25 MiB per commit.
- API and workers both call the provider, so storage must be reachable from both
  deployments.

## Entry Points and Boundaries

`git.Provider` defines repository create/delete, file list/read, commit/head,
and branch list/create/merge/delete. Repository IDs derive from
`OrganizationID` and `CanvasID`. `RepositoryProvisionerWorker` creates a
repository after `CanvasCreated`; API and agent tools read files;
#GitStagingVersioning commits and merges staged operations.

Related components: [Git Staging and Versioning](../components/git-staging-and-versioning.component.md),
[Managed Agents](../components/managed-agents.component.md), and
[Workflow Execution](../components/workflow-execution.component.md).

## System Contracts

### Key Contracts

- Paths and refs are validated by the provider contract. User operations may
  not write the reserved `.superplane` path.
- `CommitOptions.ExpectedHeadSHA` provides optimistic concurrency; a mismatched
  branch head returns `ErrExpectedHeadMismatch`.
- A commit is a batch of `FileOperation` values and has explicit author and
  message metadata.
- Repository provisioning is asynchronous. A newly created canvas can exist in
  PostgreSQL before its Git repository is ready.
- Git storage failure must not be represented as a successful stage/commit.
  Cross-store operations are not globally transactional.

### Integration Boundaries

- The storage service owns Git object integrity and filesystem durability.
- Backup/restore must coordinate repository data with PostgreSQL workflow and
  version records.
- External providers must implement the same branch and optimistic-head
  semantics expected by staging.

## Architecture Decision Records

### ADR-001: Isolate repository implementation behind `git.Provider`

**Context:** Workflow files require Git semantics while deployments may use
different storage services and tests need an in-memory implementation.

**Decision:** Keep repository operations behind a provider interface and pass
that dependency into API services, workers, runtime contexts, and agent tools.

**Consequences:** Storage can vary without changing domain callers. The shared
interface limits callers to the common repository/branch contract and leaves
cross-store consistency to orchestration.

## Open Questions

- What reconciliation repairs canvases whose repository provisioning repeatedly
  fails?
- What coordinated restore procedure guarantees PostgreSQL/Git version
  alignment?
