---
name: simplified-technical-english
description: >-
  Write user-facing and reviewer-facing technical text in ASD-STE100 Simplified
  Technical English style. Use when drafting PR descriptions, commit message
  bodies, UI copy, docs, error messages, procedures, onboarding text, or any
  product wording that must stay clear for non-native readers.
---

# Simplified Technical English (ASD-STE100)

Write clear, unambiguous technical English. Prefer STE writing rules over clever
or idiomatic phrasing.

This skill applies **STE writing rules** (grammar and style). It does **not**
embed the copyrighted STE dictionary (~900 approved words). Use simple words
with one clear meaning; use SuperPlane **technical nouns/verbs** for product
terms. For full dictionary compliance, consult the official standard:
[ASD-STE100](https://asd-ste100.org/).

Source: ASD-STE100 Issue 9 (January 2025), Standard for technical documentation.
ASD owns the STE trademark and standard; this skill is an operational distillation
for agents, not a substitute for the official document.

## When this applies

| Text | STE mode |
| --- | --- |
| PR description Summary and Test plan | Descriptive + procedural |
| Commit message **body** | Descriptive |
| UI labels, buttons, helper text, empty states, errors, confirmations | Procedural / short descriptive |
| Docs, runbooks, onboarding steps | Procedural and/or descriptive |

**Exceptions (do not rewrite these into STE):**

- Conventional Commits type prefixes (`feat:`, `fix:`, …)
- Code identifiers, API paths, proto names, log lines, and quoted error codes
- Proper nouns and product names (`SuperPlane`, `GitHub`, `PostgreSQL`)
- Issue IDs and DCO trailers

## Core rules (must follow)

### Words

1. Prefer plain words with **one meaning**. Do not use synonyms for the same idea
   in the same surface (`start` vs `begin` vs `initiate` — pick one and keep it).
2. Keep **one technical noun per concept**. Use SuperPlane vocabulary
   consistently: `canvas`, `organization`, `integration`, `component`, `trigger`,
   `run`, `approval`, `console`. Spell the product **SuperPlane**.
3. Do not use slang, idioms, humor, or jargon that the reader cannot act on.
4. Use **American English** spelling.
5. Prefer verbs for actions. Do not hide actions in nouns
   (`Create a connection`, not `Perform connection creation`).
6. Keep multi-word technical nouns to **three words or fewer** when you invent
   a phrase (`deploy approval step`, not `production deploy approval workflow step`).

### Verbs and voice

1. Prefer these forms: infinitive, **imperative**, simple present, simple past,
   simple future (`will`), past participle as adjective.
2. Avoid complex constructions: progressive (`is creating`), perfect
   (`has created`), and stacked auxiliaries when a simple form works.
3. Use the **active voice**. Use passive only in descriptive text when the actor
   is unknown.
4. Instructions and button labels use the **imperative** (`Connect GitHub`,
   `Delete the canvas`).

### Sentences

1. Write short, clear sentences. Do not omit necessary words or use contractions
   to fake brevity (`do not`, not `don't`; `cannot`, not `can't`).
2. **Procedures / instructions / test steps:** maximum **20 words** per sentence.
   One instruction per sentence unless two actions occur at the same time.
3. **Descriptions** (PR Summary, explanatory UI, notes): maximum **25 words**
   per sentence.
4. One topic per paragraph; keep paragraphs short (about six sentences or fewer).
5. Use a vertical list when a sentence would list many items or steps.
6. Give information gradually: problem or goal first, then necessary detail.

### Procedural pattern

When the reader must do something (test plan, UI steps, recovery text):

```text
If the condition is true, do the action.
```

- Put the condition first when the reader must know it before acting.
- Separate condition and command with a comma.
- Name the object and the outcome (`Delete "Deploy to production".`).

## SuperPlane technical nouns (approved for UI and PRs)

Use these as stable technical nouns. Do not invent synonyms for the same thing:

- SuperPlane
- organization
- canvas
- integration
- component
- trigger
- run
- approval
- console
- factory (Factories product surfaces)
- work order (Factories)

Code-only names (package paths, worker names, proto messages, table names) stay
out of user-facing copy.

## Before / after

**Weak (not STE):**  
`We've basically reworked how the onboarding bits kinda wire up integrations and stuff so it's nicer.`

**STE:**  
`This change updates the onboarding flow for integrations.`  
`The new flow shows connection status and the next required step.`

**Weak PR test step:**  
`Just make sure everything still works when you click around connecting GitHub and then going back.`

**STE test step:**  
`- [ ] Connect GitHub from the onboarding screen.`  
`- [ ] Confirm the status shows Connected.`  
`- [ ] Go back one step and confirm the status stays Connected.`

**Weak UI:**  
`Oops! Something went wrong on our end.`

**STE UI:**  
`SuperPlane could not connect to GitHub.`  
`Check your credentials and try again.`

## Quality checklist

- [ ] Sentences stay within 20 words (instructions) or 25 words (descriptions)
- [ ] Active voice; imperative for commands
- [ ] No contractions, slang, idioms, or hype
- [ ] One term per concept; SuperPlane vocabulary is consistent
- [ ] Lists used when steps or items would crowd a sentence
- [ ] PR/commit text explains **why** without narrative filler
- [ ] UI text states the action, object, and outcome

## Related skills

- PR titles, commit subjects, and PR structure:
  [commit-and-pr-messages](../commit-and-pr-messages/SKILL.md)
- UI microcopy patterns:
  [ui-copy](../ui-copy/SKILL.md)
