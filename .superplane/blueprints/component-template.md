# [Capability Name] Component Blueprint

## Capability Summary

Describe the reusable capability, the problem it solves, and the key contracts
flowing through it.

## Core Components

```component
name: ComponentName
container: Container Name
responsibilities:
  - Performing one focused runtime responsibility
  - Producing or consuming `ContractName`
```

Explain how `#ComponentName` collaborates with other components. State
dependency direction, what crosses the boundary, and why the relationship
exists.

```model
name: ModelName
store: Storage System
description: Purpose of the canonical domain model.
fields:
  - id: UUID (required)
constraints:
  - State a domain invariant.
```

## System Contracts

### Key Contracts

- Define invariants, consistency, ordering, idempotency, retry behavior, and
  failure semantics where relevant.

### Integration Contracts

- Define APIs, events, payloads, and composition expectations.

## Architecture Decision Records

### ADR-001: [Decision title]

**Context:** Explain why this decision is needed.

**Decision:** State what was chosen.

**Consequences:** Describe benefits, costs, risks, and implications.

## Open Questions

- Which capability or contract decisions remain unresolved?
