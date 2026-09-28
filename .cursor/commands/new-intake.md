---
description: Collect the decisions for a new factory intake, then implement that source after the human confirms the spec.
---

# New intake

You are adding a factory intake source. The human names the source, for
example Linear. You collect every product decision the implementation needs,
then you implement only after they confirm the spec.

**Use the skill `superplane-new-intake`.**

## How you work

1. If they did not name a source, ask which intake they want. Stop there.
2. Look up the integration, trigger, and any existing intake for that source.
   Say what already exists.
3. Walk the skill checklist one group at a time. Skip facts the repo already
   answers.
4. Repeat the spec and wait for confirmation.
5. Implement from the skill's file map. Reuse the closest finished intake.
   Do not add write-back, attachment copy, or merge-into-one-task unless the
   confirmed spec includes them.
