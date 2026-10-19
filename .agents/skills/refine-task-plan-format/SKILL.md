---
name: refine-task-plan-format
description: Required structure and style for the published refinement plan: title, goal paragraph, spec summary headings, optional diagram, and build spec (Scope, Approach, Acceptance, Risks). Use only when writing a refinement plan.
---

# Refine task plan format

Use this skill when you write the plan (specification) for a refined draft task.

## 6. Write the plan

Write a specification only when Clarity is 2 or higher. The short summary is for the operator. The build spec is for Start. Do not repeat the summary.

Start with '# <outcome in 8 words or fewer>'. Then write one untitled paragraph that states the goal. Do not use I think. Do not give that paragraph a heading.

Then write the spec summary with these headings in this order: '## Problem', '## Proposed outcome', and '## Constraints'. Use the same short sentences as chat. One idea per sentence. Do not name files, types, tests, or commands in the spec summary.

Good spec summary:

A duplicate action copies a backlog card into a new draft. The user can edit it before Start.

## Problem
A similar task must be typed again from scratch.

## Proposed outcome
The card gets a duplicate action. A new draft opens with the chosen fields filled.

## Constraints
Do not start the new task. Do not copy comments or run history.

Add '## Diagram' after Constraints only when one Mermaid diagram makes a UI flow or architecture easier to understand.

Follow the spec summary with the build spec. Do not repeat the goal, Problem, Proposed outcome, or Constraints. Use these headings in this order: '## Scope', '## Approach', '## Acceptance', and '## Risks'. Scope states what is in and what is out. Approach tells the implementer what to do, in order, and names existing files and seams. Acceptance includes tests and known commands. Skip Risks only when every score is 3. For Clarity 2, Risks must state what is uncertain and why. When Complexity or Verifiability is below 3, Risks must name where the implementer will need a human decision or review, and the narrower scope you proposed.

Use American English. Do not explain the repository or product. Do not write I or you in the specification.
