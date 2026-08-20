# AGENTS Improvement Spec

## Scope

This spec audits the repository guidance currently available to agents and
contributors:

- `AGENTS.md`
- `web_src/AGENTS.md`
- `.agents/skills/clean-code/SKILL.md`
- `.agents/skills/clean-code/reference.md`
- `.cursor/rules/integration-issue-conventions.mdc`
- `.cursor/rules/issue-logger-conventions.mdc`
- `.cursor/rules/ui-shadcn-forms.mdc`
- Supporting docs referenced during the audit:
  - `docs/contributing/ai-agents.md`
  - `docs/contributing/agent-tools.md`
  - `docs/contributing/pull-requests.md`
  - `docs/contributing/quality.md`

No `.ona/skills/` files were present in the repository at the time of this
audit.

## What's Good

- `AGENTS.md` gives a strong high-level map of the repository, including backend,
  frontend, infrastructure, generated-code boundaries, and PR expectations.
- The setup section correctly centers the Docker-based workflow and points agents
  at `make` targets instead of ad hoc host commands.
- The backend guidance includes specific project rules that are easy for agents
  to apply, especially protobuf enum mapping, authorization coverage, worker
  startup, generated artifacts, and migration creation.
- The database transaction section is unusually actionable: it explains the
  legacy problem, the desired `*gorm.DB`-first model API, examples, and debt
  tracking commands.
- The model API shape guidance prevents a common drift pattern: procedural helper
  functions that re-key models already loaded by callers.
- `web_src/AGENTS.md` gives concrete frontend conventions for shadcn forms,
  Storybook coverage, component state, exports, empty states, and interaction
  patterns.
- The Cursor rules capture specialized workflows that do not belong in the main
  guide: integration issue drafting, general issue logging, and shadcn form
  enforcement.
- The repository-local `clean-code` skill is well structured, broadly reusable,
  and aligns with the root `AGENTS.md` mandate to work test-first and keep code
  clear.
- Supporting docs already exist for AI-agent usage, managed agent tools, PRs,
  quality, e2e tests, integration authoring, and component design. The agent
  guidance can link to these instead of becoming a full duplicate manual.

## What's Missing

- `AGENTS.md` does not explicitly tell agents to read applicable nested guidance
  and rule files beyond `web_src/AGENTS.md`. It should include a short
  "context discovery" checklist covering nested `AGENTS.md`, `.agents/skills/`,
  `.cursor/rules/`, and relevant `docs/contributing/*` files.
- There is no conflict-resolution policy for overlapping instructions. Agents
  should know the intended precedence, for example: task request, nested
  directory guidance, root `AGENTS.md`, repo skill/rule files, then broader docs.
- The root guide does not mention the repository-local `.agents/skills/clean-code`
  skill even though its rules directly overlap with the root clean-code section.
  Add a short pointer so agents know this skill is canonical detailed guidance
  when doing implementation, refactoring, or reviews.
- There is no concise "verification matrix" by change type. The current checks
  are scattered across setup, build, frontend, protobuf, and PR sections. Agents
  would benefit from a table mapping backend, frontend, protobuf, database,
  docs-only, and issue-drafting changes to required commands.
- The frontend guide does not clearly distinguish host-side npm commands from
  repo-level Docker `make` commands. It lists `npm run build && npm run test`,
  while the root guide expects Docker-backed `make` targets.
- `web_src/AGENTS.md` does not mention the root user-facing brand spelling rule
  ("SuperPlane", not "Superplane"). Frontend work is highly likely to touch
  visible strings, so the nested guide should repeat or link this rule.
- The root guide does not mention app-agent or managed-agent backend tool
  conventions even though `docs/contributing/agent-tools.md` exists and is
  relevant to `pkg/agents/agent_tools`.
- There is no explicit guidance for authentication, authorization, tenant/org
  scoping, or secret handling in new backend work beyond local config and API
  endpoint authorization coverage.
- The generated-code section says what not to edit by hand, but it does not tell
  agents how to check generated artifacts stayed untracked. The Makefile has
  `make check.generated.artifacts`; the guide should mention it near `make
  pb.gen`.
- The database section does not mention migration verification targets that exist
  in the Makefile: `make check.db.structure` and `make check.db.migrations`.
- The issue-drafting Cursor rules are not referenced from `AGENTS.md` or
  `docs/contributing/ai-agents.md`, so non-Cursor agents may miss them.
- There is no guidance about keeping changes scoped and preserving unrelated
  user work in a dirty worktree. This is important for coding agents operating
  in shared environments.
- There is no agent-friendly troubleshooting section for Docker daemon state,
  missing containers, stale npm modules, RabbitMQ/Postgres readiness, or port
  conflicts beyond the Go module cache note.
- The root guide does not state whether agents should prefer existing project
  wrappers and Make targets over direct `go`, `npm`, `docker`, or script calls.
  The behavior is implied by the setup model but should be explicit.

## What's Wrong

- `AGENTS.md` says `make dev.setup` migrates only `superplane_dev` by default
  and requires `DEV_SETUP_DBS="superplane_dev superplane_test"` for the test DB.
  The current `Makefile` target creates and migrates both `superplane_dev` and
  `superplane_test` unconditionally.
- `web_src/AGENTS.md` says to run `npm run build && npm run test` regularly and
  later says to ensure `make check.lint.ui` and `make check.build.ui` pass. The
  preferred verification path is inconsistent. The guide should lead with the
  repo-level Docker-backed `make` targets and reserve direct npm commands for
  inside-container or local frontend-only development.
- `web_src/AGENTS.md` says "Use Task tool or check actual import files" for
  component discovery. "Task tool" is tool-specific language and may be
  meaningless outside Cursor. Replace with tool-neutral guidance such as "search
  the component exports and nearby usage."
- `web_src/AGENTS.md` duplicates several rules from `.cursor/rules/ui-shadcn-forms.mdc`
  and the root guide, but there is no single source of truth. This creates drift
  risk when shadcn, helper locations, or verification commands change.
- The root guide says the environment is entirely Docker-based and that Go and
  Node are provided by the dev container. This is directionally useful, but it
  conflates Docker Compose containers with the Dev Container. Tighten wording so
  agents understand commands run in the Compose `app` container through `make`,
  while `.devcontainer/devcontainer.json` is a separate host/editor environment.
- The root guide tells agents to leave migration `*.down.sql` files empty, but it
  does not emphasize that the down file is created by the migration target and
  should not be edited manually. This can still lead agents to hand-edit
  migration files after creation.
- `web_src/AGENTS.md` uses a generic `src/` layout example while the real
  frontend lives under `web_src/src/`. This is understandable from the nested
  file location, but agent instructions are safer when paths are explicit.
- The root guide tells agents to renumber proto fields contiguously after removing
  fields and not use `reserved`. That may be intentional for JSON conversion, but
  it is unusual protobuf advice. It should include a short "project-specific"
  warning and link to the script/doc that enforces it so agents do not "fix" it
  back to standard wire-compatibility practice.
- `docs/contributing/ai-agents.md` says `AGENTS.md` files are automatically
  loaded in Cursor as workspace rules. In this repository, Cursor-specific rules
  also exist under `.cursor/rules/`. The doc should distinguish repository
  `AGENTS.md` guidance from Cursor `.mdc` rules and say both matter.

## Improvement Spec

### 1. Update `AGENTS.md` Setup Accuracy

- Change the `make dev.setup` description to match the current `Makefile`:
  it installs npm dependencies, downloads Go modules, generates protobuf/API
  artifacts, creates both `superplane_dev` and `superplane_test`, and migrates
  both databases.
- Remove or rewrite the stale `DEV_SETUP_DBS="superplane_dev superplane_test"`
  sentence unless the Makefile is changed to support it again.
- Clarify that routine commands should be run from the repository root through
  `make`, which executes inside Docker Compose where applicable.

### 2. Add Agent Context Discovery

Add a short section near the top of `AGENTS.md`:

```markdown
## Agent Context Discovery

- Read this file before editing.
- For files under a nested directory with its own `AGENTS.md`, read that file
  too and apply the more specific guidance.
- For frontend work, read `web_src/AGENTS.md` and applicable `.cursor/rules/*`
  form/component rules.
- For implementation, refactoring, or reviews, apply `.agents/skills/clean-code`.
- For managed-agent tools, read `docs/contributing/agent-tools.md`.
- For issue-drafting workflows, read the matching `.cursor/rules/*.mdc` file.
```

Also add precedence guidance:

```markdown
When instructions overlap, prefer the most specific applicable file. If a user
request conflicts with repository safety rules, ask before proceeding.
```

### 3. Add a Verification Matrix

Add a compact table to `AGENTS.md`:

| Change type | Minimum verification |
| --- | --- |
| Go backend | `make format.go`, `make lint`, `make check.build.app`, focused `make test PKG_TEST_PACKAGES=...` |
| Shared backend behavior | Above plus broader `make test` when practical |
| Frontend | `make format.js`, `make check.lint.ui`, `make check.build.ui`, focused `make check.test.ui` when behavior changes |
| Protobuf/API | `make pb.gen`, `make check.proto.field.numbers`, `make check.generated.artifacts`, relevant backend/frontend checks |
| Database migration | `make db.migration.create NAME=<dash-name>`, `make db.migrate DB_NAME=...`, `make check.db.structure`, `make check.db.migrations` |
| Docs-only | Markdown review plus no code checks unless examples or generated docs changed |
| Issue drafts | Apply `.cursor/rules/issue-logger-conventions.mdc` or `.cursor/rules/integration-issue-conventions.mdc`; do not log without explicit approval |

### 4. Normalize Frontend Guidance

Update `web_src/AGENTS.md` to:

- Lead with repo-level verification commands:
  - `make format.js`
  - `make check.lint.ui`
  - `make check.build.ui`
  - `make check.test.ui` for behavior changes
- Move direct `npm` commands into an "inside `web_src` / local frontend loop"
  note.
- Replace "Use Task tool" with tool-neutral guidance: search exports, inspect
  nearby imports, and verify actual component APIs before inventing components.
- Make paths explicit as `web_src/src/...`.
- Repeat the user-facing brand spelling rule or link back to root `AGENTS.md`.
- Keep shadcn rules either in this file or in `.cursor/rules/ui-shadcn-forms.mdc`,
  but make one file the canonical source and link to it from the other.

### 5. Add Backend Safety Guidance

Add a backend section to `AGENTS.md` covering:

- Always preserve organization/user/canvas scoping in queries, actions, tools,
  workers, and authorization checks.
- New API endpoints must include authorization coverage in
  `pkg/authorization/interceptor.go` and focused tests where practical.
- New managed-agent tools or app actions must follow
  `docs/contributing/agent-tools.md`.
- Do not expose secrets in logs, errors, prompts, agent tool results, UI state,
  or docs examples.
- Prefer typed request validation and explicit error messages over permissive
  fallback behavior.

### 6. Tighten Database and Migration Instructions

In `AGENTS.md`:

- Make clear that migration files are created only by `make
  db.migration.create NAME=<dash-name>`.
- Say not to hand-edit the generated `.down.sql` file; leave it empty.
- Add `make check.db.structure` and `make check.db.migrations` to the database
  verification guidance.
- Keep the explicit-transaction model API guidance as-is; it is strong and
  should remain prominent.

### 7. Clarify Generated Artifacts

In `AGENTS.md`:

- Add `make check.generated.artifacts` after `make pb.gen` guidance.
- State that generated directories should remain untracked unless the project
  intentionally changes that policy.
- Add a project-specific warning around proto field renumbering so agents do not
  apply generic protobuf wire-compatibility advice.

### 8. Link Specialized Rules From Main Docs

- Add a short "Specialized Agent Workflows" section to `AGENTS.md` pointing to:
  - `.cursor/rules/integration-issue-conventions.mdc`
  - `.cursor/rules/issue-logger-conventions.mdc`
  - `.cursor/rules/ui-shadcn-forms.mdc`
  - `docs/contributing/agent-tools.md`
- Update `docs/contributing/ai-agents.md` to mention both `AGENTS.md` files and
  `.cursor/rules/*.mdc` files.

### 9. Add Dirty Worktree Guidance

Add a short safety rule to `AGENTS.md`:

```markdown
Agents may work in a dirty tree. Do not revert, overwrite, reformat, or move
unrelated user changes. Before broad formatting or generated-code commands,
check the worktree and keep the change scoped.
```

### 10. Add Troubleshooting Notes

Add concise troubleshooting entries for:

- `make dev.test.is.running` failures: run `make dev.up`.
- Docker daemon/socket unavailable in cloud environments: use the existing
  Cursor Cloud notes.
- Stale Go module cache: keep the existing `make dev.clean.go.cache` guidance.
- Stale frontend dependencies: run `make dev.setup.npm`.
- Port conflicts or server state: use `make dev.logs`, `make dev.logs.app`, and
  `make dev.down` when needed.

## Proposed Order of Work

1. Update `AGENTS.md` for setup accuracy, context discovery, verification,
   backend safety, generated artifacts, database checks, specialized workflows,
   dirty worktree safety, and troubleshooting.
2. Update `web_src/AGENTS.md` to normalize frontend commands, remove
   tool-specific wording, clarify paths, and resolve duplicate shadcn guidance.
3. Update `docs/contributing/ai-agents.md` to mention `.cursor/rules/*.mdc` and
   repository-local skills.
4. Optionally reduce duplication by making `.cursor/rules/ui-shadcn-forms.mdc`
   the canonical shadcn form rule and linking to it from `web_src/AGENTS.md`.
5. Run a docs-only verification pass:
   - Review Markdown links.
   - Confirm all referenced Make targets exist.
   - Confirm all referenced files exist.

## Acceptance Criteria

- `AGENTS.md` no longer disagrees with the current `Makefile` about
  `make dev.setup`.
- Agents can discover all applicable guidance files from the root guide without
  knowing Cursor-specific conventions.
- Backend, frontend, protobuf, migration, and docs-only verification commands
  are clear and non-contradictory.
- Frontend guidance consistently prefers repo-level Docker-backed `make` targets
  while still allowing direct npm commands for local frontend loops.
- Specialized issue-drafting and integration rules are discoverable by
  non-Cursor agents.
- The guidance explicitly protects user work in dirty worktrees.
- The guidance avoids duplicating long docs where links to existing
  `docs/contributing/*` files are enough.
