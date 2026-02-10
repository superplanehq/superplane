---
description: Generate a "what's new" changelog from merged commits over a time range (e.g. since Monday, last 5 days). Writes user-focused markdown to tmp/.
---

# Changelog

Generate a changelog of what was merged to `main` for a given time range. The output is a single markdown file in `tmp/` with new integrations, new components and triggers, improvements, security updates, and bug fixes.

**Use the skill `superplane-changelog`** for the full workflow: time range, classifying commits (new integrations vs new components vs improvements vs security vs bug fixes), format rules (no em dashes, no "We", **Integration:** components, user-focused improvements, dedicated Security section with CVE when available), and output structure. Section titles must include numeric counts for both integrations (e.g. "#### 3 new integrations") and components/triggers (e.g. "#### 12 new components and triggers").

## Input

- **Time range** (required): e.g. "since Monday", "last 5 days", "since last Friday", "from Feb 3 to now", or "since v0.6.0". If the user does not specify, ask or default to "since Monday (5 days)".

## Process

1. Determine start and end of the window. When the range is version-based (e.g. "since v0.6.0"), use **date and time**: get the tag's commit timestamp (e.g. `git log -1 --format="%cI" v0.6.0`) so the window is strictly after the tag and same-day commits before the tag are excluded.
2. Run `git log --since="<start>" --format="%h %ad %s" main` with `<start>` as a date (`YYYY-MM-DD`) or as the tag's ISO 8601 timestamp when using a version tag. Use `--date=short` or `--date=iso` as appropriate. Use the result to identify what landed in the window.
3. Classify: new integrations (whole integration new), new components/triggers only (filter by date; for existing integrations list only new components), user-facing improvements (no tech-only items), security fixes (separate section; CVE when available), bug fixes. Do not include or derive entries from `chore:` commits. Omit bug/security fixes that only affect a component or integration introduced in this changelog window.
4. Resolve component/trigger names from `pkg/integrations/` and `pkg/components/` (Labels).
5. Write `tmp/changelog_<start>_to_<end>.md` following the skill's structure and format rules.

## Output

- Path to the generated file (e.g. `tmp/changelog_2026-02-03_to_2026-02-06.md`).
- Invite the user to review and edit the file as needed.
