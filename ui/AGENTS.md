# Guidelines for AI Agents

This document outlines the guidelines for AI agents contributing to the Superplane UI repository. The goal is to ensure that AI-generated code is consistent, high-quality, and easy to maintain.

## Component Structure

- **Location**: Place component implementations under `src/components/<name>/index.tsx`. If the shadcn CLI generates files inside `src/components/ui`, move the implementation into the component folder and remove the duplicate `ui` entry.
- **Storybook Stories**: Story files should be named `[component-name].stories.tsx` (e.g., `button.stories.tsx`).
- **Storybook Titles**: Story titles should be in the format `components/[Component Name]` (e.g., `components/Button`).

## shadcn Workflow

- **Install**: Run `npx shadcn@latest add <component>` to scaffold primitives.
- **Relocate**: Move generated files from `src/components/ui` into the corresponding `src/components/<name>` directory and update imports accordingly.
- **Separator Dependency**: Use existing components (e.g., `../separator`) when shadcn templates depend on primitives, rather than importing from `/ui`.

## Prop Design

- **Clarify Contracts**: Before implementation, confirm or document the expected props—including required fields, optional sections, and icon behavior.
- **Dynamic Icons**: When a prop supplies a Lucide icon via slug (e.g., `list-filter-plus`), map it to the Lucide export and provide a safe fallback.
- **Optional UI**: Only render event types, filters, hover cards, etc., when the corresponding props are provided.

## Storybook Best Practices

- **Use Controls**: Use `argTypes` to create controls for props, enabling interactive docs.
- **Consolidate Stories**: Prefer a single `Default` story with controls to switch variants.
- **Distinct Scenarios**: Create extra stories only for meaningfully different use cases.
- **Disable Irrelevant Controls**: Disable controls that shouldn’t apply to a story.
- **Respect static assets**: Do not modify existing integration logos/SVGs unless intentional.

## Code Style & Validation

- **Imports**: Use relative paths between components (e.g., `../button`).
- **Clarity**: Write concise, clear code; comment only when logic is non-obvious.
- **Testing**: After component/story updates, run `npm run storybook -- --ci --smoke-test` to ensure the Storybook build succeeds.

By following these guidelines, AI agents can produce consistent, high-quality contributions to this project.
