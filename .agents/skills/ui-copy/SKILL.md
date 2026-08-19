---
name: ui-copy
description: >-
  Use when writing, editing, reviewing, or replacing UI copy, UX writing,
  microcopy, button labels, link text, form labels, helper text, empty states,
  error messages, confirmation dialogs, onboarding text, notifications, or
  user-facing product wording — including when designing or implementing UI.
  Applies ASD-STE100 Simplified Technical English style.
---

# UI Copy

Write interface copy as part of the design, not as decoration after the layout
is done. The product is speaking to someone who is trying to complete a task.

Adapted from Operately's [ui-copy skill](https://github.com/operately/operately/blob/main/.agents/skills/ui-copy/SKILL.md)
for SuperPlane product vocabulary.

**Language:** Write all user-facing strings in ASD-STE100 Simplified Technical
English style. Follow
[simplified-technical-english](../simplified-technical-english/SKILL.md)
together with this skill. If UX tone and STE conflict, choose the clearer STE
wording.

## Workflow

1. Identify the user's current goal, screen state, and likely concern.
2. Find nearby existing copy and match the product vocabulary, casing, tense, and tone.
3. Draft the clearest STE version first. Prefer plain, specific, task-oriented language over personality.
4. Tighten the copy until every word has a job (≤20 words for instructions, ≤25 for descriptions).
5. Check that buttons, links, errors, and helper text set accurate expectations.
6. If changing existing copy, scan adjacent screens or components for terms that should stay consistent.

## Principles

- Treat words as UI. Real copy belongs in early designs, prototypes, and implementation; do not rely on placeholder text when the wording affects comprehension, trust, or layout.
- Clarity beats brevity and personality. Short is good only when it remains specific. Do not use cute, clever, or branded wording when it can hide the action.
- Write for the user in this moment. Ask what they are trying to do, what they need to know now, what can wait, and whether they need reassurance.
- Be honest and concrete. Avoid marketing adjectives, hype, vague promises, and copy that tells users how to feel.
- Use STE plain language. Avoid jargon, unexplained acronyms, idioms, contractions, internal names, and technical error codes unless the user can act on them.
- Prefer active voice, imperative or simple present, and specific verbs: `Create`, `Save`, `Archive`, `Invite`, `Download`, `Connect`.
- Keep copy scannable. Use short sentences, front-load important words, and reveal advanced detail only when needed.
- Keep vocabulary consistent. Do not alternate between terms like `canvas`, `workflow`, and `pipeline` unless the product treats them as different concepts.
- Match the platform and interaction. Use `tap` for touch surfaces and `click` only where pointer interaction is the assumption.
- Do not use humor in UI copy. Never use humor in high-friction, high-risk, error, payment, privacy, or destructive flows.

## Product Vocabulary

Some names exist only in code, schemas, and internal discussion. Never put them in UI copy — placeholders, labels, empty states, errors, confirmations, notifications, or help text.

- The product name is **SuperPlane** (capital S, capital P). Never write `Superplane` or `superplane` in user-facing copy.
- Prefer established product terms already used in the UI: `canvas`, `organization`, `integration`, `component`, `trigger`, `run`, `approval`, `console`. Do not invent synonyms for the same concept.
- Do not expose internal package paths, proto message names, worker names, or DB table names in UI strings.
- When in doubt, match copy on a nearby screen rather than inventing a new term.

## Buttons

Buttons should describe the action or outcome. A user should not be surprised by what happens after clicking.

- Prefer `Create canvas` over `Submit`.
- Prefer `Save and continue` over `Next` when the next step matters.
- Prefer `Delete integration` and `Keep integration` over `OK` and `Cancel` in destructive confirmations.
- Prefer `Connect GitHub` over `Continue` when the next step is a specific connection.
- Avoid vague commitment language when a lower-pressure action is accurate.

Use short helper text near a high-commitment button when it answers a likely concern:

- `You can change this later.`
- `Only organization admins can see this.`
- `This will not start a run yet.`

## Links

Links can carry more context than buttons, but they still need to be descriptive when scanned out of context.

- Prefer `View run history` over `Learn more`.
- Prefer `Download the CSV report` over `Download`.
- Prefer `See organization permissions` over `Details`.

Use `Learn more` only when the destination is genuinely broad and the surrounding heading already makes the topic obvious.

## States And Messages

For empty states, say what is missing and offer the next useful action:

- Weak: `No results.`
- Better: `No matching canvases. Try a different filter or create a canvas.`

For errors, explain what happened in user terms and give a recovery path:

- Weak: `System error #2234.`
- Better: `We could not connect to GitHub. Check your credentials and try again.`

For confirmations, name the object, consequence, and escape path:

- Weak: `Are you sure?`
- Better: `Delete "Deploy to production"? This removes the canvas for everyone.`

For success messages, confirm the completed action without hype:

- Weak: `Awesome! Your amazing update was successful.`
- Better: `Canvas updated.`

## Review Checklist

- Does the copy help the user complete the current task?
- Is the primary action named with a specific verb?
- Could the user understand it without reading surrounding body text?
- Does the wording obey STE limits (≤20 words for instructions, ≤25 for descriptions)?
- Is any word internal, technical, vague, promotional, or trying too hard?
- Are contractions, slang, idioms, or humor absent?
- Are terms, casing, and point of view consistent with nearby UI?
- Is detail progressively disclosed instead of shown all at once?
- Does the copy still fit small screens and common localization expansion?
- Is the product name spelled **SuperPlane** wherever it appears?

## Source Distillation

This skill distills principles from:

- Julie Chabin, `Good UI can't fix bad copy`
- John Zeratsky, `Five principles for great interface copywriting`
- Nick Babich, `16 Rules of Effective UX Writing`
- Tobias van Schneider, `Writing UX copy for buttons and links`
