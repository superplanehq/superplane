# AI Agent Component Skill Awareness

## Overview

This PRD defines how SuperPlane's AI chat assistant should discover and use component-specific
`SKILL.md` files while helping users build and modify workflows.

When the assistant plans to use a component, it should retrieve that component's skill document and
incorporate the guidance into its reasoning, suggestions, and generated canvas operations.

## Problem Statement

The AI assistant can currently suggest components and wiring patterns, but its output quality depends
on generalized knowledge and may miss SuperPlane-specific implementation details. This creates issues:

- Suggestions can omit required configuration details for a component.
- Generated wiring can conflict with component-specific constraints.
- Assistant guidance can drift from documented best practices.
- Users may need manual correction cycles before a workflow is valid.

Without component skill grounding, the assistant is less reliable and less aligned with how components
are intended to be used in SuperPlane.

## Goals

1. Ground AI chat responses in component-specific `SKILL.md` guidance whenever a component is involved.
2. Improve first-attempt correctness of generated component configurations and channel mappings.
3. Reduce back-and-forth caused by missing component constraints or best practices.
4. Keep the behavior transparent so users understand that component skills are being applied.

## Non-Goals

- Replacing global product policies or organization-level AI safety rules.
- Introducing autonomous execution without user review and apply confirmation.
- Requiring every component to have a `SKILL.md` before AI can assist.
- Building long-term persistent memory of skill content in v1.

## Primary Users

- **Workflow Builders**: Users creating workflows with AI assistance.
- **Integration Authors**: Users validating that AI follows documented component usage.
- **Support/Enablement Teams**: Teams troubleshooting why AI made a specific suggestion.

## User Stories

1. As a workflow builder, when AI suggests a component, I want it to follow that component's documented guidance.
2. As a user, when AI proposes a node configuration, I want required fields and common pitfalls to be handled correctly.
3. As an integration author, I want updates to a component's `SKILL.md` to improve AI behavior for that component.
4. As a user, if no component skill exists, I still want useful AI assistance with a clear fallback behavior.

## Functional Requirements

### Skill Discovery

- The assistant must identify candidate components relevant to a user prompt before generating a final plan.
- For each candidate component, the assistant must attempt to locate the component's `SKILL.md`.
- Skill lookup should support known component skill locations used by SuperPlane component/integration assets.

### Skill Retrieval and Parsing

- On successful lookup, the assistant must load skill content and extract actionable guidance, including:
  - Required configuration fields and defaults.
  - Input/output expectations and channel mapping notes.
  - Constraints, caveats, and recommended patterns.
- If a skill file is missing, unreadable, or invalid, the assistant must continue using standard reasoning with fallback rules.

### Response Grounding Behavior

- When skill content is available, assistant responses must prioritize skill guidance over generic assumptions.
- Proposed canvas operations should reflect constraints described in the relevant skills.
- If multiple components are involved, the assistant should merge guidance across all found skills and resolve conflicts safely.

### Transparency in Chat

- Responses that depend on component skills should include a short, user-friendly indication that component guidance was applied.
- The assistant should mention missing skill coverage only when it materially affects confidence or recommendation quality.

### Fallback Behavior

- If no skill exists for a component, the assistant should:
  - Continue to provide best-effort guidance.
  - Avoid overconfident claims.
  - Encourage user review of proposed configs before apply.

### Performance and Freshness

- Skill lookup and ingestion should add minimal latency to chat responses.
- Skill content should be refreshed often enough that recent updates are reflected without requiring app redeploys.
- Implement lightweight caching with clear invalidation behavior for v1.

### Safety and Data Handling

- Skill ingestion must not expose secrets or sensitive runtime values in prompts or responses.
- Retrieved skill content must be treated as advisory guidance and still pass standard operation validation before apply.

## UX Requirements

- Users should see no additional setup for components that already provide `SKILL.md`.
- Chat responses should remain concise; skill usage indicators should be subtle and non-disruptive.
- Error states for missing/unreadable skill files should not block normal chat flow.

## Implementation Scope (v1)

- Enable component skill lookup during AI chat request processing.
- Apply skill grounding to assistant explanation text and proposed canvas operations.
- Add basic telemetry for:
  - Components detected per prompt.
  - Skill files found vs. missing.
  - Skill-grounded proposal apply success rate.
- Keep scope limited to runtime retrieval and in-request grounding (no persistent skill embeddings in v1).

## Out of Scope (v1)

- Automated linting/validation of `SKILL.md` content quality.
- A UI for browsing or editing component skill documents.
- Fine-grained per-paragraph source citations in chat messages.
- Organization-specific custom skill overrides.

## Acceptance Criteria

1. For prompts involving known components with available `SKILL.md`, AI responses and proposals reflect component-specific guidance.
2. For prompts involving multiple components, AI behavior incorporates guidance from each available component skill.
3. When component skills are missing or unreadable, AI still responds with best-effort guidance and does not fail the chat request.
4. Skill-grounded proposals continue to pass existing validation before apply; invalid operations are blocked as today.
5. Chat latency impact from skill lookup remains within acceptable product thresholds.
6. Telemetry records skill discovery outcomes (found/missing/error) for each relevant component.

## Success Metrics

- Increase in first-attempt valid proposal apply rate for prompts that involve components with skills.
- Reduction in user edits required after AI-generated component configuration.
- Reduction in support cases where AI ignored documented component constraints.
- Coverage metric: percentage of component-involving prompts where at least one component skill was successfully applied.

## Risks and Mitigations

- **Risk:** Skill documents become stale or inconsistent.  
  **Mitigation:** Define ownership and update expectations for component skill content; add basic freshness checks.

- **Risk:** Conflicting guidance across multiple component skills.  
  **Mitigation:** Apply conservative conflict resolution and prioritize safe, valid operations.

- **Risk:** Latency regressions from repeated skill lookups.  
  **Mitigation:** Use request-scoped caching and lightweight refresh strategies.

- **Risk:** Over-reliance on imperfect skill content.  
  **Mitigation:** Keep existing validation and user approval flow as the source of truth before apply.

## Open Questions

1. What are the canonical repository paths for component `SKILL.md` files in all integration/component types?
2. Should missing `SKILL.md` for high-usage components trigger internal quality alerts?
3. Do we want optional "why this suggestion" citations that reference specific skill sections in future versions?
4. Should organizations eventually be able to layer private guidance on top of component skills?
