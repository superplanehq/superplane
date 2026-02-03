---
description: Research, classify, draft, and optionally log a general SuperPlane issue (bug, enhancement, feature, papercut) to tmp/pm_logger and GitHub.
---

# Issue Logger

You are helping the user log a general SuperPlane issue (not an integration issue). They will describe an improvement, bug, or request in natural language. Your job is to research and understand it in SuperPlane context, classify the issue type, suggest priority (P1–P4), propose a title and body in `tmp/pm_logger`, and optionally create the issue on GitHub with the correct label and Board Priority.

**Use the skill `superplane-issue-logger`** for the full workflow: research and understanding, issue type classification (bug / enhancement / feature / papercut), priority rules (P1–P4), body templates, draft location (`tmp/pm_logger/`), screenshot handling, and optional GitHub MCP steps (create issue, type label, Board, Priority; suggest video after creation). Follow the **issue-logger-conventions** rule when creating or editing files in `tmp/pm_logger`.

## Input

- The user's message: a natural-language description of the improvement, bug, or request (e.g. "The Copy button in the toolbar is truncated", "Add bulk select for canvas nodes", "Small typo in the settings header").
- If the description is vague, ask for: what exactly is wrong or desired, where in the app it happens, expected vs actual (for bugs), and steps to reproduce (for bugs).
- If they describe multiple items, ask which one to capture first or create separate drafts.

## Process

1. **Research and clarify**: Use docs/contributing and docs.superplane.com as needed. If the description is unclear, prompt the user before classifying.
2. **Classify**: Map to exactly one of **bug**, **enhancement**, **feature**, **papercut** using the skill's definitions (e.g. breaks flow → bug; annoying but workable → papercut).
3. **Suggest priority**: Propose P1–P4 with a brief rationale using the skill's rules. **Ask the user to confirm or change** before writing the draft.
4. **Draft**: Write a **short title (max 40 characters)** and body to `tmp/pm_logger/<slug>.md` using the correct short template (bug, papercut, enhancement, feature). Title: one or two concrete concepts; not a full list of details, not too vague (e.g. "layout and icons" not "polish", not "long names/IDs rendering, icons, responsive width"). **Do not put Priority in the body** (it is set via the Board only). For bug or papercut, if the user didn't provide a screenshot, ask in chat; if they pasted one in chat, note that they can paste it into the issue after creation (MCP cannot upload images to the issue body).
5. **Optional — log to GitHub**: **Only after the user has verified the draft and explicitly approved** (e.g. "looks good", "log it"). Do not create the issue until then. When approved: read the draft, create the issue via GitHub MCP, add the type label (`bug` / `enhancement` / `feature` / `papercut`), add to SuperPlane Board (project 2), set Priority (P1–P4) via Board fields, then suggest they attach a video to the issue.

## Output

- After drafting: show the suggested title, type, priority, and path to the draft file; **ask the user to verify the draft and confirm** before logging. Do not log to GitHub until the user says the draft looks OK (e.g. "looks good", "log it").
- After logging (if requested): confirm issue number and link; remind them to attach a video to the issue if helpful.

## Constraints

- One issue per draft; one type label per issue. **Never log to GitHub before the user has verified the draft and approved.**
- Do not set Integration Status on the Board (that is for integration issues only).
- Create issues sequentially when logging multiple drafts to avoid rate limits.
- Keep drafts short and efficient; use the skill's templates, not long boilerplate. Title = max 40 characters, one or two concrete concepts (not a full list, not too vague); no Priority line in body.
