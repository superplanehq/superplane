---
description: Review a component implementation against SuperPlane standards.
---

You are reviewing a SuperPlane component implementation. Follow the rules in `.cursor/commands/component-review.rules.md` in order, one by one.

Input:
- Use the user's selection or the provided component name/path.
- If the component is ambiguous, ask one clarifying question before proceeding.

Process:
1) Identify whether this is a core component (`pkg/components/...`) or an integration component (`pkg/integrations/...`).
2) Locate the component's `Name()`, `Label()`, `Description()`, `Documentation()`, `Icon()`, `Color()`, `ExampleOutput()`, and `Configuration()`.
3) Evaluate each rule from the rules file in order.

Output:
- For every rule, output a subtitle with the rule name, then a single line with the result:
   - Subtitle format: `### <short rule name>`
   - Result line (OK): `OK`
   - Result line (NOT OK): `NOT OK: <evidence>`
- If a rule fails evidence must cite concrete files and identifiers (function names, constants, files).
- If a rule fails, include a short, actionable hint after the evidence on the same line.
- After listing all rules, add a brief "Summary" section listing only failed rules (names + evidence + actionable hint)

Constraints:
- Do not skip rules.
- Provide OK/NOT OK only (no "N/A").
- Keep the output concise and scannable.

Extensibility:
- Anyone can add rules by editing `.cursor/commands/component-review.rules.md`.
- The command must always read the rules file and apply any new rules in order.
