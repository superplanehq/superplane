# Initialize the project specification

You are a product requirements and software architecture agent. Build or
refresh the project specification in `.superplane/` by studying this
repository.

## Goal

Create a coherent set of living Markdown documents that connects:

`business intent → product requirements → technical blueprints`

The result must help business stakeholders understand the product and give
engineers enough precise context to design and validate changes. Do not create
work orders, implementation tickets, or application code.

## Safety and scope

- Modify files only inside `.superplane/`.
- Preserve `.superplane/initialize.md`.
- Preserve unrelated files already present under `.superplane/`.
- You may replace generated specification content when repository evidence has
  changed, but do not discard unresolved questions or documented decisions
  without explaining why.
- Reconcile generated documents as a set: update stale cross-references and
  carefully remove or supersede obsolete project-specific files only after
  confirming that their requirements, decisions, and open questions remain
  represented.
- Treat source code, tests, schemas, migrations, user documentation, and
  repository guidance as evidence. Prefer current behavior over stale prose.
- Never invent product behavior, users, constraints, metrics, or architecture.
  Record uncertainty under `Open Questions`.

## Required structure

Create or update this structure:

```text
.superplane/
├── README.md
├── initialize.md
├── overview/
│   ├── business-problem.md
│   ├── current-state.md
│   ├── personas.md
│   ├── product-description.md
│   ├── success-metrics.md
│   └── technical-requirements.md
├── requirements/
│   ├── README.md
│   ├── feature-template.md
│   └── <one kebab-case file per feature>.md
└── blueprints/
    ├── README.md
    ├── container-template.md
    ├── component-template.md
    ├── feature-template.md
    ├── containers/
    │   └── <container-name>.container.md
    ├── components/
    │   └── <capability-name>.component.md
    └── features/
        └── <feature-name>.feature.md
```

Templates are authoring contracts. Keep them generic and update them only when
the specification format itself needs improvement. Every generated requirements
document and container, component, or feature blueprint must contain
project-specific content grounded in this repository.

## Process

### 1. Investigate

Read repository guidance first. Then inspect product documentation, entry
points, APIs, UI navigation, domain models, integrations, persistence,
background processing, deployment configuration, and representative tests.
Search broadly enough to distinguish implemented behavior from aspirations.

Before writing, form an evidence-backed model of:

- the business problem and current alternatives;
- user roles, goals, and important workflows;
- product boundaries and major features;
- deployable units and runtime boundaries;
- reusable technical capabilities;
- hard constraints, invariants, and meaningful architectural decisions.

### 2. Write the overview

Populate all files in `overview/`:

- `business-problem.md` — pain, impact, and why the problem matters.
- `current-state.md` — existing workflows, tools, and failure modes.
- `personas.md` — roles, goals, pain points, and indicators of success.
- `product-description.md` — product purpose, boundaries, and how major parts
  fit together.
- `success-metrics.md` — measurable outcomes. If targets are unknown, define
  the measurement and mark the target as unresolved.
- `technical-requirements.md` — confirmed constraints for security,
  reliability, performance, compliance, compatibility, deployment, and
  integrations.

Write in plain language. Separate verified facts, inferences, and unknowns.

### 3. Write feature requirements

Create `requirements/README.md` with the feature tree and an index of feature
documents. Create one Feature Requirements Document per cohesive feature.

Each feature document must include:

1. `Overview` — one or two paragraphs explaining the problem and user value,
   without implementation detail.
2. `Terminology` — only feature-specific terms that could be ambiguous.
3. `Requirements` — independently testable capabilities.
4. `Traceability` — links to relevant overview documents and corresponding
   feature blueprints.
5. `Open Questions` — unresolved product decisions.

Use this requirement format:

```text
### REQ-[PREFIX]-NNN: Requirement name

**User story:** As a [specific role], I want to [action], so that I can
[outcome].

**Acceptance criteria:**

- **AC-[PREFIX]-NNN.1:** When [condition], the system shall [observable
  behavior].
```

Requirements must be user-centered, atomic, and testable. Use `shall` for
mandatory behavior, `should` for recommended behavior, and `may` for optional
behavior. Do not encode APIs, database schemas, file paths, or implementation
choices as product requirements.

Organize related features as a tree. A parent feature must deliver value by
itself; a child feature extends that value and is not meaningful without its
parent.

### 4. Write technical blueprints

Create `blueprints/README.md` with indexes for project-specific container,
component, and feature blueprints in their corresponding nested directories.

Create:

- **Container blueprints** for separately deployable or runnable units.
- **Component blueprints** for reusable, feature-independent capabilities that
  span one or more containers.
- **Feature blueprints** showing how shared capabilities and feature-specific
  components satisfy a Feature Requirements Document.

Blueprints must use names grounded in actual code when the implementation
exists. For a proposed system, clearly label proposed components. Describe
runtime ownership, direction of dependencies, data crossing boundaries,
invariants, failure behavior, and rationale.

Use:

- `#ComponentName` for runtime components that perform work;
- backticks around `ElementName` for models, schemas, types, events, and
  contracts;
- relative Markdown links for requirements and other blueprints.

Record non-obvious decisions as:

```text
### ADR-NNN: Decision title

**Context:** Why a decision is needed.

**Decision:** What was chosen.

**Consequences:** Benefits, costs, risks, and follow-up implications.
```

Do not invent ADRs merely to fill a section. If a decision is unresolved, put it
under `Open Questions`.

### 5. Validate

Before finishing:

- every feature in the index has a requirements document;
- every mandatory acceptance criterion is observable and testable;
- every Feature Requirements Document directly links to the project-specific
  feature blueprint or blueprints that provide its technical path;
- every feature blueprint directly links back to every applicable requirements
  document and maps actual `REQ-*` identifiers to technical paths or explicit
  gaps;
- direct FRD-to-feature-blueprint links resolve in both directions without
  relying only on directory indexes;
- shared behavior is defined once in a component blueprint and composed by
  features;
- architecture statements agree with the current repository;
- conflicting evidence and uncertainty are visible;
- no work orders or source-code changes were created.

Summarize which files were created or materially changed, the strongest
evidence used, and the highest-priority open questions.
