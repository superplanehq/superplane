# Project Specification

This directory is the source of truth for the project's product intent,
requirements, and technical design.

## Structure

- `initialize.md` — reusable prompt for rebuilding and refreshing the
  specification from repository evidence.
- `overview/` — business context, users, product scope, success measures, and
  technical constraints.
- `requirements/` — flat, project-specific Feature Requirements Documents
  expressed as user outcomes and testable acceptance criteria, plus their index
  and generic authoring template.
- `blueprints/` — project-specific technical designs organized under
  `containers/`, `components/`, and `features/`, plus indexes and generic
  authoring templates.

## Authoring order

1. Establish the product overview.
2. Derive a feature tree and feature requirements.
3. Describe the technical system with blueprints.
4. Verify direct, bidirectional traceability between each Feature Requirements
   Document and its feature blueprint or blueprints.

Requirements define what the product must do and why. Blueprints define how the
system satisfies those requirements. Unknowns must be recorded explicitly
rather than filled with unsupported assumptions.

To initialize or refresh the specification, give the contents of
`initialize.md` to an agent with access to the repository.
