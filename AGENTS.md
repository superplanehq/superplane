# Repository Guidelines

**Read this file in full.** Do not truncate it with `head`, `sed`, or a partial Read.

SuperPlane is an open-source automation engine for AI-driven engineering, with durable execution, approvals, and an operational UI.
The stack uses Go (gRPC and REST/OpenAPI), TypeScript/React/Vite, PostgreSQL, and RabbitMQ.
Use **SuperPlane** in all user-facing text.

## Generated files and build system

`make pb.gen` generates these paths. Git ignores them:

- `pkg/protos/` — Go code from `protos/*.proto`
- `pkg/openapi_client/` — Go SDK for the CLI
- `web_src/src/api-client/` — TypeScript SDK for the UI
- `api/` — OpenAPI/swagger spec

Never hand-edit or commit generated files. Edit `protos/`, then run `make pb.gen`.
A clean clone lacks these files until `make dev.setup` or `make pb.gen` runs.
They do not appear in `git status` or `git diff`.
Do not search Git history or the working tree for them as tracked changes.

Development is entirely Docker-based. Build, lint, test, and run commands execute in containers through Make and Docker Compose.
Use the Make targets below. Do not probe for raw Docker Compose filenames or `.github/workflows`.
CI uses Semaphore (`.semaphore/`) and runs `make dev.setup`, which includes `pb.gen`.

## Read before the relevant task

- Setup, local runners, factory dispatch, or tool/cache failures: [Development Quickstart](CONTRIBUTING.md#development-quickstart).
- Local third-party services, factory GitHub/Sentry apps, or hosted OpenRouter: [service connections](docs/contributing/connecting-to-3rdparty-services-from-development.md).
- Cursor Cloud VMs: [cloud setup](docs/contributing/cursor-cloud.md).
- Codebase navigation: [directory structure](docs/contributing/architecture.md#directory-structure).
- Backend extensions: [backend extension rules](docs/contributing/architecture.md#backend-extension-rules).
- Add or activate runtime dependencies, including removal of flags or guards: [deployment dependencies](docs/contributing/runtime-configuration.md#deployment-dependencies).
- `pkg/models` changes or new database access: [model rules](pkg/models/AGENTS.md).
- Built-in automation settings, including UI: [factory rules](pkg/grpc/actions/factories/AGENTS.md). Preserve settings through admin reset and add regression coverage.
- UI work: [frontend rules](web_src/AGENTS.md), [UI copy](.agents/skills/ui-copy/SKILL.md), and [STE](.agents/skills/simplified-technical-english/SKILL.md).
- New components/triggers: [implementation](docs/contributing/component-implementations.md) and [design](docs/contributing/component-design.md).
- E2E authoring: [E2E guide](docs/contributing/e2e-tests.md).
- Dev server profiling: [profiling guide](docs/contributing/profiling.md).
- Write, change, review, or sweep tests: load [test-audit](.agents/skills/test-audit/SKILL.md).
- Commit messages or PR titles/descriptions: [commit-and-pr-messages](.agents/skills/commit-and-pr-messages/SKILL.md).
- Documentation, UI copy, or other reviewer/user-facing technical text: [STE](.agents/skills/simplified-technical-english/SKILL.md).

## Setup and required checks

Run initial setup in order:

1. `make dev.up` — build images and start dependency containers.
2. `make dev.setup` — install dependencies, generate code, and migrate `superplane_dev`.
3. `make dev.server` — start the API, Vite, and Docker Fleet Manager.

Read the quickstart before setup or recovery. Re-run `make dev.setup` after proto, Go module, or frontend dependency changes.

Run the relevant checks after edits and before submission:

| Change | Required commands, in order |
| --- | --- |
| Go | `make format.go`, `make lint`, `make check.build.app`, `make test` |
| JS/TS | `make format.js`, `make check.lint.ui`, `make check.build.ui` |
| Protos | `make pb.gen`, `make check.proto.field.numbers` |
| Admin API routes | `make check.admin.routes` |

Do not run `go build ./...`; `scripts/` contains multiple `main` packages. Use `make check.build.app`.
Run UI lint locally before opening a PR. CI fails when the ESLint budget grows.
Code generation requires the running `app` container from `make dev.up`.
After removing proto fields, renumber the remaining fields contiguously. Do not leave gaps or `reserved` markers.
These protos serve JSON conversion, not wire compatibility.

Test commands:

- Backend: `make test`; targeted: `make test PKG_TEST_PACKAGES=./pkg/workers`.
- E2E: `E2E_TEST_PACKAGES=./test/e2e/workflows make test.e2e`.
  Single test: `make test.e2e.single FILE=test/e2e/foo_test.go LINE=19`.
  Semaphore runs E2E only on `main`; local commands work on all branches.
- UI (Bun + Happy DOM): `make check.test.ui`.
  Targeted: `make check.test.ui FILES=src/lib/duration.spec.ts`. `FILES` paths are relative to `web_src/`.

## Database safeguards

Never create or edit migration files by hand. Use `make db.migration.create NAME=<name>` with dashes, not underscores.
Leave `*.down.sql` empty; we do not write rollbacks.
After adding a migration, run `make db.migrate DB_NAME=<DB_NAME>` with the `app` container running.
Use `superplane_dev` or `superplane_test` for `DB_NAME`.

Never drop or recreate local databases without an explicit user request. This includes:

- `make db.delete`
- `make db.recreate.all.dangerous`
- `make dev.setup.no.cache`
- `make dev.pr.clean.checkout`
- `dropdb` or `DROP DATABASE`

`make db.migrate.all` applies pending migrations and rewrites `db/structure.sql` without dropping data.
Do not pair it with `db.delete`.
If migration fails (for example, `no migration found for version`), stop and ask.
Do not recreate `superplane_dev` as a workaround.

## Coding and file protections

- Work test-first by default. Keep names clear, functions focused, side effects explicit, control flow shallow, and error handling useful.
  Prefer early returns over `else` blocks.
- Test observable behavior, not incidental presentation. Follow the test-audit authoring gate and retention rules.
  Preserve visibility, accessibility, theme, and other functional coverage.
  Prefer accessible roles and stable test IDs. Assert exact UI text only when wording is the behavior under test.
  Avoid repeated copy, CSS, layout, or DOM assertions. Prefer real behavior over duplicate private-helper or generated-source assertions.
  Before removing overlapping coverage, confirm the retained test catches the same regression and runs in CI.
- Go tests end with `_test.go`. Base specific test timestamps on `time.Now()`, not absolute `time.Date` values.
- Go: use `any` instead of `interface{}`. Use `slices.Contains` or `slices.ContainsFunc` for slice membership.
  Avoid type suffixes such as `*Str` or `*UUID` in variable names.
- Name non-obvious UI state decisions. Extract conditions that combine states or exceptions into named functions.
  Comment when the product reason is unclear.
- Do not create or use `web_src/src/utils/*` or `utils.ts`.
  Put shared non-React helpers in `web_src/src/lib/` and reusable React logic in `web_src/src/hooks/`.
- Never commit real secrets or a populated `.env`. `.env.example` and `.env.multi-instance.example` are templates.
- Do not edit `web_src/node_modules/` or the Go module cache under `tmp/go/`.

## Commits and pull requests

Follow the required writing skills above. Use short sentences, active voice, imperative instructions, and stable terminology.
Do not use contractions, slang, or idioms.

- PR titles require `feat:`, `fix:`, `chore:`, or `docs:`.
  Use an imperative, capitalized summary after the prefix, with no trailing period.
- Use a short STE `Summary` only. Explain the problem, motivation, and user/system impact rather than narrating the diff.
  Do not add a Test plan or a checklist of completed checks. Avoid empty or placeholder bodies.
- Commit subjects use the same imperative style, approximately 50 characters.
  Separate the body with a blank line, wrap at 72 characters, and explain motivation and trade-offs.
- All commits require a DCO sign-off (`Signed-off-by: Name <email>`). Use `git commit -s`.
- Run the relevant commands in [Setup and required checks](#setup-and-required-checks) before submission.
