# Relevant DB tables: workflows, executions, secrets

## Entity-relationship (ASCII)

```
┌─────────────────────┐
│   organizations     │
├─────────────────────┤
│ id (PK)             │
│ name                │
│ ...                 │
└──────────┬──────────┘
           │ 1
           │
           │ N
┌──────────▼──────────┐         ┌─────────────────────────┐
│     workflows       │         │       secrets           │
├─────────────────────┤         ├─────────────────────────┤
│ id (PK)             │         │ id (PK)                 │
│ organization_id (FK)│         │ name                    │
│ name                │         │ domain_type             │  ← 'org'
│ edges (jsonb)       │         │ domain_id               │  ← org id (no FK)
│ nodes (jsonb)       │         │ provider                │
│ ...                 │         │ data (bytea)            │  ← encrypted k/v
└──────────┬──────────┘         │ created_by, ...         │
           │ 1                 └─────────────────────────┘
           │ N
┌──────────▼──────────────────────────────────────────────────┐
│                    workflow_nodes                              │
├───────────────────────────────────────────────────────────────┤
│ workflow_id (PK,FK)                                            │
│ node_id (PK)                                                  │
│ name, type, state                                              │
│ ref (jsonb)              ← component/trigger/blueprint ref     │
│ configuration (jsonb)    ← NODE config template ({{ expr }})  │
│ parent_node_id (FK self)                                       │
│ app_installation_id, webhook_id, ...                            │
└──────────┬───────────────────────────────────────────────────┘
           │
           │ 1
           ├──────────────────────────────┬─────────────────────────────┐
           │ N                             │ N                           │ N
┌──────────▼──────────────┐  ┌────────────▼─────────────┐  ┌───────────▼────────────────┐
│   workflow_events       │  │ workflow_node_queue_items │  │ workflow_node_executions    │
├─────────────────────────┤  ├───────────────────────────┤  ├────────────────────────────┤
│ id (PK)                 │  │ id (PK)                   │  │ id (PK)                    │
│ workflow_id (FK)        │  │ workflow_id (FK)          │  │ workflow_id (FK)           │
│ node_id (FK)            │  │ node_id (FK)              │  │ node_id (FK)               │
│ channel, data (jsonb)   │  │ root_event_id (FK)       │  │ root_event_id (FK)         │
│ execution_id (FK) ──────┼──│ event_id (FK)             │  │ event_id (FK)             │
│ custom_name             │  │ created_at                │  │ previous_execution_id (FK) │
└─────────────────────────┘  └───────────────────────────┘  │ parent_execution_id (FK)  │
           ▲                           │                      │ state, result, ...        │
           │                           │                      │ configuration (jsonb) ◀───┼── RESOLVED config (secrets end up here today)
           │                           │                      └───────────┬──────────────┘
           │                           │                                  │ 1
           │                           │                                  │ N
           │                           │                      ┌────────────▼─────────────┐
           │                           └─────────────────────│ workflow_node_execution_ │
           │                                                 │         kvs              │
           │                                                 ├──────────────────────────┤
           │                                                 │ id (PK)                  │
           │                                                 │ execution_id (FK)        │
           │                                                 │ key, value               │
           │                                                 │ workflow_id, node_id     │
           │                                                 └──────────────────────────┘
           │
           │  workflow_node_executions.event_id
           │  workflow_node_executions.root_event_id
           └──────────────────────────────────────
```

## Table summaries

| Table | Purpose | Config / secrets |
|-------|---------|------------------|
| **organizations** | Org metadata | — |
| **workflows** | Canvas (workflow) definition; `nodes`/`edges` are denormalized JSON | — |
| **workflow_nodes** | Per-node row: type, ref, **configuration** (template with `{{ }}` and optional secret refs) | **configuration** = node config template (what user edited). Not resolved. |
| **workflow_events** | Events (trigger output, node output); link to execution that produced them | `data` = event payload (no config) |
| **workflow_node_queue_items** | Pending work for a node (event_id, root_event_id); consumed when building execution | No config stored |
| **workflow_node_executions** | One row per node run: state, result, **configuration** (resolved at build time) | **configuration** = resolved config used for this run. Today this is where secret values are persisted. |
| **workflow_node_execution_kvs** | Key-value store for execution (e.g. lookup by key) | key/value only |
| **secrets** | Org-scoped secrets; `domain_type`='org', `domain_id`=org id; **data** = encrypted blob (e.g. `{"api_key":"..."}`) | **data** = encrypted; never stored in workflow/execution tables |

## Where config lives

- **Node (template):** `workflow_nodes.configuration` — expressions and refs, not resolved. Safe to expose (no secret values if we use refs only).
- **Execution (resolved):** `workflow_node_executions.configuration` — today this is the fully resolved config, including secret values. This is what gets returned by list-executions API and is the “don’t store / don’t expose” target when moving to refs or resolve-at-runtime.

## Relationships (FKs)

- `workflows.organization_id` → `organizations.id`
- `workflow_nodes.workflow_id` → `workflows.id`
- `workflow_nodes.parent_node_id` → `workflow_nodes.node_id` (same workflow)
- `workflow_events.workflow_id` → `workflows.id`
- `workflow_events.(workflow_id, node_id)` → `workflow_nodes.(workflow_id, node_id)`
- `workflow_events.execution_id` → `workflow_node_executions.id`
- `workflow_node_queue_items.(workflow_id, node_id)` → `workflow_nodes.(workflow_id, node_id)`
- `workflow_node_queue_items.event_id` → `workflow_events.id`
- `workflow_node_queue_items.root_event_id` → `workflow_events.id`
- `workflow_node_executions.(workflow_id, node_id)` → `workflow_nodes.(workflow_id, node_id)`
- `workflow_node_executions.root_event_id` → `workflow_events.id`
- `workflow_node_executions.event_id` → `workflow_events.id`
- `workflow_node_executions.previous_execution_id` → `workflow_node_executions.id`
- `workflow_node_executions.parent_execution_id` → `workflow_node_executions.id`
- `workflow_node_execution_kvs.execution_id` → `workflow_node_executions.id`

**secrets** has no FK to workflows; it is scoped by `(domain_type, domain_id)` where `domain_type = 'org'` and `domain_id` is the organization id.
