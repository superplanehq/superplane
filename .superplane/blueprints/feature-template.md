# [Feature Name] Feature Blueprint

## Feature Summary

Summarize the user outcome and link to the corresponding
[Feature Requirements Document](../requirements/).

## Component Blueprint Composition

Identify the shared component blueprints this feature uses. Explain how the
feature configures and connects their components, including dependency
direction and exchanged contracts. Do not redefine shared components here.

## Feature-Specific Components

```component
name: FeatureSpecificComponent
container: Container Name
responsibilities:
  - Performing behavior unique to this feature
  - Collaborating with `#SharedComponent` using `ContractName`
```

Explain the feature-specific runtime flow and why these components are not
shared capabilities.

## System Contracts

### Key Contracts

- Define feature invariants, correctness rules, authorization rules, and
  reliability semantics.

### Integration Contracts

- Define feature-specific APIs, events, payloads, and composition expectations.

## Requirement Coverage

- **REQ-[PREFIX]-001:** Explain the technical path that satisfies this
  requirement.

## Architecture Decision Records

### ADR-001: [Decision title]

**Context:** Explain why this decision is needed.

**Decision:** State what was chosen.

**Consequences:** Describe benefits, costs, risks, and implications.

## Open Questions

- Which implementation or composition decisions remain unresolved?
