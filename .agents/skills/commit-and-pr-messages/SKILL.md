---
name: commit-and-pr-messages
description: >-
  Write Git commit messages and pull request titles/descriptions using the
  Chris Beams / Tim Pope conventions (subject/body split, ~50-char subject,
  imperative mood, why-not-how body) plus ASD-STE100 Simplified Technical
  English for bodies and PR descriptions. Use when creating commits, drafting
  or editing PR descriptions, writing PR titles, or when the user asks for help
  with commit/PR wording, changelogs from commits, or git history style.
---

# Commit and PR Messages

Write history that future readers can scan, search, and trust. A diff shows
*what* changed; the message must explain *why*.

Based on [How to Write a Git Commit Message](https://cbea.ms/git-commit/)
(Chris Beams) and [A Note About Git Commit Messages](https://tbaggery.com/2008/04/19/a-note-about-git-commit-messages.html)
(Tim Pope), adapted for SuperPlane's Conventional Commits + DCO requirements.

**Language:** Write commit bodies, PR titles (after the type prefix), and PR
descriptions in ASD-STE100 Simplified Technical English style. Follow
[simplified-technical-english](../simplified-technical-english/SKILL.md)
before you finalize wording.

## When to use

- Creating or amending a commit message
- Opening or updating a pull request (title and body)
- Reviewing whether a PR description is useful to reviewers

## Commit messages — seven rules

1. **Separate subject from body with a blank line.**
2. **Limit the subject line to ~50 characters** (72 hard max). Prefer the
   Conventional Commits prefix used by this repo (`feat:`, `fix:`, `chore:`,
   `docs:`) plus a short imperative summary; keep the whole subject readable.
3. **Capitalize the subject line** after the optional type prefix
   (`feat: Add canvas console filters`).
4. **Do not end the subject line with a period.**
5. **Use the imperative mood in the subject line** — as if completing:
   *If applied, this commit will ________.*
   Prefer `Fix`, `Add`, `Remove`, `Refactor` over `Fixed`, `Adds`, `Fixing`.
6. **Wrap the body at 72 characters.**
7. **Use the body to explain what and why, not how.** The code shows how.
   Cover motivation, prior behavior, trade-offs, and non-obvious side effects.

### Model commit

```text
Capitalized, short (50 chars or less) summary

More detailed explanatory text, if necessary. Wrap it to about 72
characters or so. The blank line after the subject is mandatory when a
body is present.

Explain the problem this commit solves and why this approach was chosen.
Leave mechanical how-details to the diff unless the approach is surprising.

Further paragraphs come after blank lines.

- Bullet points are okay

Resolves: #123
```

### SuperPlane commit extras

- Include a DCO trailer: `Signed-off-by: Name <email>` (use `git commit -s`).
- One logical change per commit when practical; if the subject is hard to
  write, the commit may be doing too much.
- Issue/PR references belong at the end of the body, not stuffed into the
  subject.

## Pull request titles

PR titles are the public subject line for a whole change set.

- Follow Conventional Commits with a release-type prefix CI enforces:
  `feat:`, `fix:`, `chore:`, or `docs:`.
- After the prefix, write an imperative, capitalized summary with no trailing
  period — same mood test as commits.
- Keep the title scannable; put nuance in the body, not a novel in the title.

**Good:** `feat: Add empty-state copy for connected integrations`  
**Weak:** `feat: Updated some stuff for onboarding`  
**Weak:** `Fixed the bug.`

## Pull request descriptions

Treat the PR body like an expanded commit body for reviewers and future
maintainers. Lead with purpose; do not narrate the diff file-by-file.
Write the Summary and Test plan in STE
([simplified-technical-english](../simplified-technical-english/SKILL.md)).

### Required shape

```markdown
## Summary

<2–4 short STE sentences or bullets: what problem this solves and why this
change exists. Max 25 words per descriptive sentence. Focus on motivation
and user/system impact, not implementation tourism.>

## Test plan

- [ ] <STE procedural step: max 20 words, one instruction, imperative>
- [ ] <Edge cases or regressions worth checking>
```

### Writing rules for the Summary

- **STE first.** Short sentences, active voice, no contractions, no slang.
- **Why over how.** State the before/after behavior and the reason for the
  change. Mention approach only when it is non-obvious or risky.
- **Present / imperative orientation.** Describe what the PR *does*
  (`Adds…`, `Fixes…`, `Removes…`), not a diary of what you did yesterday.
- **Scannable.** Short paragraphs or bullets; no wall of implementation notes
  that duplicate the diff.
- **Honest scope.** Call out follow-ups, known gaps, or intentional
  non-goals when relevant.
- **Link context.** Reference issues (`Closes #123`) when they exist; do not
  invent ticket IDs.

### Avoid

- Subject-only PRs with an empty or placeholder body when the change needs
  context
- Restating every file touched (`Updated Foo.tsx, Bar.go, …`)
- Past-tense changelog dumping with no problem statement
- Marketing or hype language

### Quick quality check

Before submitting a PR description, confirm:

- [ ] Title completes: *If merged, this PR will ________.*
- [ ] Summary explains **why**, not a file list
- [ ] Summary and Test plan obey STE sentence limits and style
- [ ] Test plan has actionable checks
- [ ] No trailing period / non-imperative mush in the title
- [ ] Body would still make sense months later without the author present
