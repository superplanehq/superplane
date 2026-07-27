# PostgreSQL Container Blueprint

## Container Summary

PostgreSQL is SuperPlane’s authoritative transactional store. GORM-backed model
code owns access to accounts, organizations, RBAC metadata, integrations,
secrets, workflows, live versions, staged files, runtime events, queue items,
executions, runs, webhooks, agent sessions, and operational metadata.

Evidence: [`db/structure.sql`](../../../db/structure.sql),
[`db/migrations/`](../../../db/migrations/),
[`pkg/models/`](../../../pkg/models/), and
[`pkg/database/`](../../../pkg/database/).

## Infrastructure

- Deployments may use an external database or the chart’s single-replica
  PostgreSQL `StatefulSet` and persistent volume
  ([`database.yaml`](../../../release/superplane-helm-chart/helm/templates/database.yaml)).
- The single-host release uses PostgreSQL 17.5; chart versions are configurable.
  Runtime compatibility, not a single image tag, is the contract.
- API and worker pools are configured independently with `DB_POOL_SIZE`.
  Statement and idle-in-transaction timeouts are deployment settings.
- Schema changes are applied by ordered files under `db/migrations`; model code
  must not assume a migration is optional.

## Entry Points and Boundaries

- API handlers use request-scoped `database.DB(ctx)` where migrated; workers and
  legacy paths also use `database.Conn()`.
- Workers coordinate with row locks, transactions, leases, and PostgreSQL
  advisory locks. The database therefore participates in concurrency control,
  not only persistence.
- Git repository contents are not stored here. `workflows`,
  `workflow_versions`, and `workflow_staged_files` connect domain identity and
  effective workflow state to Git-backed files.
- Encrypted values are stored as bytes; cryptographic keys remain deployment
  secrets outside PostgreSQL.

Related components: [Workflow Execution](../components/workflow-execution.component.md),
[Authentication and RBAC](../components/authentication-and-rbac.component.md),
and [Git Staging and Versioning](../components/git-staging-and-versioning.component.md).

## System Contracts

### Key Contracts

- `Organization` IDs scope tenant-owned rows; callers must include that scope
  unless a deliberately unscoped worker lookup is used.
- Transactions define atomic domain transitions such as event routing, workflow
  publication, webhook claims, and request completion.
- State transitions are the idempotency fence for asynchronous consumers.
  Repeated RabbitMQ delivery must not repeat completed database transitions.
- Foreign keys, unique indexes, and model validation jointly enforce identity;
  source code must not rely on application checks alone.
- A committed database transition can precede a failed RabbitMQ publish, so
  polling/reconciliation behavior is part of the reliability contract.

### Integration Boundaries

- Backups must be coordinated with Git storage if a restore is expected to
  preserve exact workflow repository/version references.
- External database operators own replication, failover, encryption at rest,
  point-in-time recovery, and connection limits.

## Architecture Decision Records

### ADR-001: Keep runtime orchestration state relational

**Context:** Workflow execution requires atomic graph routing, queue ownership,
tenant scoping, and queryable run history.

**Decision:** Persist canonical control-plane and execution state in PostgreSQL
and use database transactions/locks for coordination.

**Consequences:** Correctness can be expressed with transactional state
transitions. Database availability and lock contention directly affect both API
and worker progress.

## Open Questions

- What recovery point and recovery time objectives apply to supported
  installations?
- Which remaining legacy model functions should be migrated to explicit
  request-scoped `*gorm.DB` parameters?
