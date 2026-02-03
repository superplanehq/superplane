---
name: superplane-issue-logger
description: When researching, classifying, drafting, or logging general SuperPlane issues (bug, enhancement, feature, papercut). Use for natural-language issue descriptions, tmp/pm_logger drafts, priority suggestions, and optional GitHub MCP logging with labels and Board Priority.
---

# SuperPlane Issue Logger

Use this skill when a user describes an improvement, bug, or request in natural language. You will: (1) research and understand it in SuperPlane context, (2) classify issue type, (3) suggest priority (P1–P4), (4) propose title and body in `tmp/pm_logger`, and (5) optionally create the issue on GitHub with correct labels and Board Priority.

Reference: [docs/contributing/](docs/contributing/) (issue-tracking, quality, pull-requests, etc.), [https://docs.superplane.com/](https://docs.superplane.com/) when relevant.

---

## 1. Research and understanding

- **Context**: Use docs/contributing/ (issue-tracking, quality, pull-requests, integrations, component-design) and, when relevant, https://docs.superplane.com/ (product behavior, UX).
- **Clarification**: If the description is vague (e.g. "improve the canvas" or "fix the button"), prompt the user for: what exactly is wrong or desired, where in the app it happens, expected vs actual behavior, and (for bugs) steps to reproduce.
- **Scope**: One issue per draft. If the user describes multiple items, ask which one to capture first or create separate drafts.

---

## 2. Issue type classification

Map the user's description to exactly one of:

| Type            | Definition                                                                      |
| --------------- | ------------------------------------------------------------------------------- |
| **bug**         | Something isn't working as expected (incorrect behavior, failure, regression). |
| **enhancement** | Small/medium improvement to an existing feature already in the app.             |
| **feature**     | Completely new feature or functionality.                                       |
| **papercut**    | Small issue, bug, or inconsistency; minor improvement that doesn't break flows.  |

- **Bug vs papercut**: If it breaks a flow or blocks usage → **bug**. If it's annoying or inconsistent but workable → **papercut**.
- **Enhancement vs feature**: Improves something that already exists → **enhancement**. Adds net-new capability → **feature**.

Align with [issue-tracking](docs/contributing/issue-tracking.md) (Bugs, Papercuts, Enhancements views on the SuperPlane Board).

---

## 3. Priority suggestion (P1–P4)

Suggest a single priority and brief rationale. Priorities are **P1–P4**. Obtain field and option ids via GitHub MCP `list_project_fields` for project 2 when logging to GitHub.

**Rules:**

- **P1**: Critical bug (product or core flow largely unusable, e.g. login broken, canvas won't load). Or flagship feature that defines a new product capability or segment.
- **P2**: Major bug with significant impact; major enhancement that greatly improves core functionality; feature that majorly improves UX or adds clearly valuable new functionality for devops/development workflows.
- **P3**: Normal bug (reproducible, meaningful impact, workaround exists); meaningful enhancement or feature that improves UX or workflows in a clear but not "major" way.
- **P4**: Papercuts; small bugs with minimal impact; nice-to-have enhancements or features.

Always **show the suggested priority and rationale** and ask the user to confirm or change before writing the draft (and before logging to GitHub).

---

## 4. Title and body in `tmp/pm_logger`

- **Location**: `tmp/pm_logger/` (create directory if missing). One file per issue draft (e.g. `bug-toolbar-copy-button.md`, `papercut-header-truncation.md`).
- **Title**: Short — **max 40 characters**. One or two concrete concepts; not a full list of details, not too vague. Good: "Search canvas modal: layout and icons" (34), "Payload modal: drag-to-resize" (30). Too detailed: "Search canvas modal: long names/IDs rendering, icons, responsive width". Too vague: "Search canvas modal: polish". Put in the file as `### Title` (or a clear header) so the log step can reuse it.
- **Body**: Use the short template below for the chosen type. Keep drafts efficient; avoid long sections or empty boilerplate. **Do not add Priority to the body** — Priority is set only via the Board Project field when logging to GitHub.
- **Screenshot**: Screenshots are optional. For bug or papercut, if the user didn't provide one, ask in chat if they can share a screenshot (paste or attach). **Screenshots pasted in chat cannot be uploaded to the issue via MCP** (GitHub API accepts only text in the body). If the user shared a screenshot in chat: note in the body that they can paste it into the issue after creation (GitHub's issue/comment UI supports paste-to-upload), or omit and suggest they add it once the issue is open. Do not add an empty "Screenshot" section; do not put raw image data in the body.

### Bug (short)

```markdown
**Description:**  
Brief statement of what's wrong.

**Steps to reproduce:**  
1. …  
2. …

**Expected / Actual:**  
Expected: … Actual: …
```

(Screenshot only if user provided one in chat.)

### Papercut (minimal)

```markdown
**What:** One sentence — small issue or inconsistency.  
**Where:** Screen/component and element.
```

(Screenshot only if user provided one.)

### Enhancement (short)

```markdown
**Problem:** Existing feature + what to improve (1–2 sentences).  
**Current vs proposed:** One line each.  
**Use case:** Who benefits and when.
```

### Feature (short)

```markdown
**Describe the request:**  
1–2 sentences: what to add or build.

___

**Describe your use-case:**  
Why this is needed (user pain or opportunity; 1–2 sentences).

___

**Describe functionality:**
- Bullet 1
- Bullet 2
- …
```

**Instructions**: (1) Choose the template by type. (2) Fill from the user's description and any clarification. (3) Keep text concise; leave placeholders only where the user didn't provide info.

---

## 5. Log issue via GitHub MCP (optional)

**Do not log the issue to GitHub until the user has confirmed that the draft (title, body, type, priority) looks OK.** Show the draft, ask for verification, and only when the user explicitly approves (e.g. "looks good", "log it", "create the issue") proceed to create the issue.

When the user has confirmed and asks to "log" or "create" the issue on GitHub:

1. **Read draft**: From `tmp/pm_logger/<file>.md` get title and body. Strip `### Title` and the title line from the body when sending to API. Strip any `**Priority:**` line from the body — Priority is set only via the Board.
2. **Create issue**: `issue_write` (method `create`) with `owner`/`repo` (e.g. `superplanehq`/`superplane`), `title`, `body`.
3. **Labels**: Set **one** type label: `bug`, `enhancement`, `feature`, or `papercut` (match repo labels; if repo uses different names, e.g. `kind: bug`, use those).
4. **Board**: Add item to SuperPlane Board (project 2) via `projects_write` (`add_project_item` with `item_type: issue`, `issue_number`, `item_owner`, `item_repo`).
5. **Priority**: Get project field id and option id for "Priority" via `list_project_fields` (project 2). Then `update_project_item` with `item_id` (from `list_project_items` for the new issue) and `updated_field` set to the chosen Priority (P1, P2, P3, or P4).

Do **not** set Integration Status (that's for integration issues). Create issues sequentially to avoid rate limits. Optionally keep a small progress note in `tmp/pm_logger/log-progress.md` and, after user confirmation, delete only the progress file (keep draft files unless user asks to remove them).

**After creating the issue:** Suggest the user attach a video (e.g. screen recording) to the issue once it's created—by adding a comment with a link or uploading—to help with triage and implementation.

---

## Summary checklist

- [ ] Research context (docs/contributing, docs.superplane.com); ask for clarification if vague.
- [ ] Classify type: bug, enhancement, feature, or papercut.
- [ ] Suggest P1–P4 and rationale; get user confirmation.
- [ ] Write draft to `tmp/pm_logger/<slug>.md` using the correct template; ask for screenshot if bug/papercut and user didn't provide one.
- [ ] If user wants to log: **only after user confirms the draft is OK** — create issue via GitHub MCP, add type label, add to Board, set Priority; then suggest user attach video to the issue.
