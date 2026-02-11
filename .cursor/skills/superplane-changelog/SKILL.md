---
name: superplane-changelog
description: When generating a SuperPlane changelog from merged commits. Use for "what's new" summaries with new integrations, new components/triggers, improvements, security updates, and bug fixes. Output is user-focused markdown in tmp/.
---

# SuperPlane Changelog

Use this skill when the user wants a changelog of what was merged to `main` over a given time range (e.g. "since Monday", "last 5 days", "since last Friday"). Produce a single markdown file in `tmp/` with a consistent structure and tone.

---

## 1. Determine time range

- **User may say**: "since Monday", "last 5 days", "since last Friday", "Feb 3 to now", "since v0.6.0", or a specific date.
- **Compute**: Start and end of the window. Use **date and time** (not just date) when the start is a version tag so that same-day commits before the tag are excluded.
  - **Date-only ranges** (e.g. "since Monday", "Feb 3 to now"): Start = date at midnight, end = today. For "last 5 days" use Monday to Friday; for "since last Friday" use that Friday through today.
  - **Version-tag ranges** (e.g. "since v0.6.0"): Start = **exact commit timestamp of the tag** (e.g. `git log -1 --format="%cI" v0.6.0` for ISO 8601). End = now or a chosen end date. This ensures commits that landed the same calendar day but before the tag are not included.
- **Git**: Use `git log --since="<start>" --format="%h %ad %s" main` where `<start>` is:
  - For date-only: `YYYY-MM-DD` (e.g. `2026-02-03`). Use `--date=short` in the format.
  - For version-tag: the tag's commit timestamp in ISO 8601 (e.g. `2026-02-01T15:30:00+00:00`). Use `--date=iso` if you need to compare times. Only include in the changelog items whose commit/merge date is **strictly after** the start when using a tag.

---

## 2. Classify what landed

From commit messages and dates:

- **Exclude `chore:` commits (mandatory).** Do not list or derive any changelog entry from commits whose subject starts with `chore:` or `chore(...):`. This applies to every section: do not add an improvement, integration, component, or any other bullet based on a chore commit, even if the change seems user-facing (e.g. "Allow running multiple instances" is still a chore and must be omitted). When classifying what landed, skip chore commits entirely; only use `feat:`, `fix:`, `docs:` (for user-facing doc changes), and similar non-chore prefixes as sources for changelog entries.
- **New integrations**: Integrations that were **fully added** in the window (base integration registered + first components). Example: SendGrid, Jira. Do **not** count standalone components (e.g. SSH is a component under `pkg/components/ssh`, not an integration).
- **New components and triggers**: Only components/triggers that **first appeared in the time window**. If an integration already existed, list only the new component(s) (e.g. GitHub: Get Release). If the integration is new, list all its components and triggers. Use commit timestamps (date and time) to exclude anything that landed before the start of the window (e.g. when the window is "since v0.6.0", exclude commits with timestamp on or before the tag's commit time, so same-day commits before the tag are excluded).
- **Improvements**: User-facing product changes from non-chore commits only (e.g. RBAC, Secrets, integrations UX). Exclude internal/technical items (e.g. "Component/Trigger Cleanup()", "listing integration resources with additional parameters", Cursor skills). Describe each improvement in user-oriented terms: what the user can do, what problem it solves, or what benefit they get (e.g. "Define roles and permissions and control what each user can do" rather than "Permission guard in the UI").
- **Security**: Vulnerability fixes and security-related changes from the same commit range. Look for commits that mention "security", "SSRF", "CVE", "vulnerability", "auth", "injection", "XSS", "sanitiz", etc. Include a dedicated **Security** section whenever at least one such fix is present. Do not list a security fix if it only affects a component or integration that was introduced in this changelog window.
- **Bug fixes**: Fixes and reliability improvements from the same commit range (excluding security fixes, which go under Security). Keep in "Bug Fixes" even if somewhat technical. Do not list a fix if it only affects a component or integration that was introduced in this changelog window (e.g. "fix: AWS ECR timestamp" when ECR was added in the same window).

To resolve component/trigger names and which integration they belong to, use `pkg/integrations/*/` and `pkg/components/*/`: check each integration's `Components()` and `Triggers()` and their `Label()` / `Name()` (e.g. `aws.go` for AWS, `ecr/`, `codeartifact/`).

---

## 3. Format rules (strict)

- **No em dashes (—)**. Use colons or parentheses instead (e.g. **RBAC**: description).
- **No "We" language**. Use direct, neutral phrasing (e.g. "Role-based access control." not "We introduced role-based access control.").
- **New integrations section**: List only integration names, one per line (e.g. SendGrid, Jira).
- **New components section**: Use **Integration:** Component1, Component2, ... One line per integration or standalone component (e.g. **GitHub:** Get Release; **SSH:** Run commands on remote hosts).
- **Improvements**: Each bullet is **Bold label**: Short, user-oriented description. Write from the user's perspective: what they can do, what problem it solves, or what benefit they get. Avoid implementation jargon (e.g. "permission guard", "payload limit"); prefer outcome and capability (e.g. "Control what each user can do in your organization", "Secrets can be used in the SSH component to store private keys"). No "We".
- **Security**: Dedicated section (use only when there are security-related commits). Each bullet: include **CVE identifier** when available (e.g. CVE-2024-12345), then a short description of the vulnerability or fix. If no CVE, use "Fixed: " plus description (e.g. "Fixed: SSRF protection added to HTTP requests"). Same tone as rest of changelog; no em dashes.
- **Bug fixes**: Each bullet starts with "Fixed: " then a short description. Do not list security fixes here; they go under Security. Omit fixes that only apply to components or integrations that are new in this changelog.

---

## 4. Output structure

Write a single file to `tmp/changelog_YYYY-MM-DD_to_YYYY-MM-DD.md` (or similar) with this structure:

- **Section titles must include the numeric count** for both integrations and components (e.g. "#### 3 new integrations", "#### 12 new components and triggers"). Count each integration as 1. For components and triggers, count each component or trigger as 1 (e.g. one line "**GitHub:** Get Release, On Release" is 2).

```markdown
# SuperPlane Changelog (Feb X-Y, YYYY)

## What's new since [Monday], [Month Day], YYYY (X days)

#### N new integrations

   - IntegrationA
   - IntegrationB

#### M new components and triggers

   - **IntegrationA:** Component1, Component2, Trigger1
   - **IntegrationB:** Component1
   - **Standalone:** Description (e.g. **SSH:** Run commands on remote hosts)

#### Improvements

   - **RBAC**: Role-based access control. Define roles and permissions...
   - **Secrets**: Create, update, and delete organization secrets...
   - (etc.)

#### Security

   - CVE-YYYY-NNNNN: Short description of vulnerability and fix (when CVE exists).
   - Fixed: Short description of security fix (when no CVE).
   (Omit this section entirely if no security-related commits in the window.)

#### Bug Fixes

   - Fixed: Short description
   - Fixed: ...
```

- Use three spaces before list bullets for indentation under each #### heading.
- Replace N and M with the actual counts. N = number of integrations listed. M = total number of components and triggers (each component or trigger counts as 1, even when several are on one line). Counts must match the listed items and the chosen time window.

---

## 5. Workflow summary

1. Ask for or infer time range (e.g. "Monday to now" = 5 days; "since v0.6.0" = after the tag's commit timestamp).
2. Run `git log --since="<start>" --format="%h %ad %s" main` with `<start>` as date (`YYYY-MM-DD`) or as the tag's commit timestamp in ISO 8601 when the range is version-based. Use `--date=short` or `--date=iso` as needed. Optionally inspect merge dates for key PRs.
3. Identify new integrations (whole new integration only), new components/triggers (per integration, only in window), improvements (user-facing only; never derived from chore commits), security fixes (dedicated section; separate from bug fixes), and bug fixes. Do not include or derive any entry from `chore:` or `chore(...):` commits in any section.
4. Resolve labels from code: `pkg/integrations/<name>/` and `pkg/components/` for component/trigger names.
5. Write `tmp/changelog_<range>.md` following the structure and format rules above.
6. Tell the user the file path and that they can review or edit it.

---

## 6. Reference example

See `tmp/changelog_2026-02-01_to_2026-02-06.md` (or the latest similar file in `tmp/`) for a concrete example of the desired style and structure.
