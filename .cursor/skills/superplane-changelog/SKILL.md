---
name: superplane-changelog
description: When generating a SuperPlane changelog from merged commits. Use for "what's new" summaries with new integrations, new components/triggers, improvements, security updates, and bug fixes. Output is user-focused markdown in tmp/.
---

# SuperPlane Changelog

Use this skill when the user wants a changelog of what was merged to `main` over a given time range (e.g. "since Monday", "last 5 days", "since last Friday"). Produce a single markdown file in `tmp/` with a consistent structure and tone.

---

## 1. Determine time range

- **User may say**: "since Monday", "last 5 days", "since last Friday", "Feb 3 to now", or a specific date.
- **Compute**: Start date (e.g. last Monday = start of week) and end date (today). For "last 5 days" use Monday to Friday; for "since last Friday" use that Friday through today.
- **Git**: Use `git log --since="YYYY-MM-DD" --format="%h %ad %s" --date=short main` to list commits. Only include in the changelog items whose merge/commit date falls **on or after** the start date.

---

## 2. Classify what landed

From commit messages and dates:

- **New integrations**: Integrations that were **fully added** in the window (base integration registered + first components). Example: SendGrid, Jira. Do **not** count standalone components (e.g. SSH is a component under `pkg/components/ssh`, not an integration).
- **New components and triggers**: Only components/triggers that **first appeared in the time window**. If an integration already existed, list only the new component(s) (e.g. GitHub: Get Release). If the integration is new, list all its components and triggers. Use commit dates to exclude anything that landed before the start date (e.g. Cloudflare DNS records merged Feb 1 are excluded if the window is "Monday Feb 3 to now").
- **Improvements**: User-facing product changes (RBAC, Secrets, Bounty Program, integrations UX, list vs expression, multiple instances). Exclude internal/technical items (e.g. "Component/Trigger Cleanup()", "listing integration resources with additional parameters", Cursor skills).
- **Security**: Vulnerability fixes and security-related changes from the same commit range. Look for commits that mention "security", "SSRF", "CVE", "vulnerability", "auth", "injection", "XSS", "sanitiz", etc. Include a dedicated **Security** section whenever at least one such fix is present.
- **Bug fixes**: Fixes and reliability improvements from the same commit range (excluding security fixes, which go under Security). Keep in "Bug Fixes" even if somewhat technical.

To resolve component/trigger names and which integration they belong to, use `pkg/integrations/*/` and `pkg/components/*/`: check each integration's `Components()` and `Triggers()` and their `Label()` / `Name()` (e.g. `aws.go` for AWS, `ecr/`, `codeartifact/`).

---

## 3. Format rules (strict)

- **No em dashes (—)**. Use colons or parentheses instead (e.g. **RBAC**: description).
- **No "We" language**. Use direct, neutral phrasing (e.g. "Role-based access control." not "We introduced role-based access control.").
- **New integrations section**: List only integration names, one per line (e.g. SendGrid, Jira).
- **New components section**: Use **Integration:** Component1, Component2, ... One line per integration or standalone component (e.g. **GitHub:** Get Release; **SSH:** Run commands on remote hosts).
- **Improvements**: Each bullet is **Bold label**: Short, user-focused description. No implementation details. No "We".
- **Security**: Dedicated section (use only when there are security-related commits). Each bullet: include **CVE identifier** when available (e.g. CVE-2024-12345), then a short description of the vulnerability or fix. If no CVE, use "Fixed: " plus description (e.g. "Fixed: SSRF protection added to HTTP requests"). Same tone as rest of changelog; no em dashes.
- **Bug fixes**: Each bullet starts with "Fixed: " then a short description. Do not list security fixes here; they go under Security.

---

## 4. Output structure

Write a single file to `tmp/changelog_YYYY-MM-DD_to_YYYY-MM-DD.md` (or similar) with this structure:

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
   - **Bounty Program**: Get paid for building integrations. See [link]...
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
- Counts (N new integrations, M new components and triggers) must match the listed items and the chosen time window.

---

## 5. Workflow summary

1. Ask for or infer time range (e.g. "Monday to now" = 5 days).
2. Run `git log --since="<start-date>" --format="%h %ad %s" --date=short main` and optionally inspect merge dates for key PRs.
3. Identify new integrations (whole new integration only), new components/triggers (per integration, only in window), improvements (user-facing only), security fixes (dedicated section; separate from bug fixes), and bug fixes.
4. Resolve labels from code: `pkg/integrations/<name>/` and `pkg/components/` for component/trigger names.
5. Write `tmp/changelog_<range>.md` following the structure and format rules above.
6. Tell the user the file path and that they can review or edit it.

---

## 6. Reference example

See `tmp/changelog_2026-02-01_to_2026-02-06.md` (or the latest similar file in `tmp/`) for a concrete example of the desired style and structure.
