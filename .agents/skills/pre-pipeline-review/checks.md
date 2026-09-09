# Fast checks matrix

Use only `make` targets. Source of truth for CI jobs:
[`.semaphore/semaphore.yml`](../../../.semaphore/semaphore.yml) and the
"After editing" section of [AGENTS.md](../../../AGENTS.md).

Run a row when **any** listed path pattern matches the scoped diff. Deduplicate
targets across rows. Skip a row when nothing in that column changed.

## Run when paths match

| Touched paths | Targets (in order) |
| --- | --- |
| `*.go` under `cmd/`, `pkg/`, `test/` (non-generated) | `make format.go` → `make lint` → `make check.build.app` → `make check.format.go` |
| `web_src/**` (TS/JS/CSS, not `web_src/src/api-client/`) | `make format.js` → `make check.format.js` → `make check.fast.security` → `make check.lint.ui` → `make check.build.ui` |
| `protos/**` | `make pb.gen` → `make check.proto.field.numbers` (never hand-edit generated output) |
| `pkg/models/**` | Also: `make check.models.tx.debt` |
| `pkg/grpc/actions/**` | Also: `make check.grpc.actions.status` |
| New gRPC / API endpoints | Confirm coverage in `pkg/authorization/interceptor.go` (manual review; no Make target) |
| Components, triggers, example payloads, or configuration field scripts / related packages | `make check.components.docs`, `make check.example.payloads`, `make check.configuration.fields` as applicable |
| `db/migrations/**` or migration-related scripts | `make check.db.migrations` (never `db.delete` / recreate) |
| Any non-docs code change | `make check.generated.artifacts` |

### Notes

- **App container**: all `exec`-based targets need a running `app` container
  (`make dev.test.is.running`). If down: `make dev.up` only.
- **Go build**: use `make check.build.app` only. Do not run `go build ./...`
  (`scripts/` has multiple `main` packages).
- **Protos**: after `pb.gen`, generated trees stay gitignored; a clean
  `git status` for those paths is expected. Do not commit them.
- **Fast security**: `make check.fast.security` is a static scan of tool
  configs, known dropper markers, and npm install hooks. Run it before
  `make check.lint.ui`. It does not load `eslint.config.js`.
- **ESLint**: `make check.lint.ui` fails when the budget grows. Fix violations;
  do not update the baseline unless the user explicitly requests it.
- **DB structure**: `make check.db.structure` is CI-only for this skill unless
  migrations or `db/structure.sql` are in the diff and the user wants that
  check. Prefer `check.db.migrations` for migration PRs.

## Skip list (not "easy")

Do **not** run unless the user explicitly asks:

| Target | Why skip |
| --- | --- |
| `make test` / `make test.coverage.autoparallel` | Slow backend unit suite |
| `make test.e2e` / `make test.e2e.autoparallel` | Slow E2E suite |
| `make check.test.ui` / `make check.test.ui.shard` | Slow frontend unit shards |
| `make check.build.storybook` | Slow Storybook build |
| `make check.lint.ui.knip` | Dead-file check; slower and often noisy |

## Suggested command batches

**Go-only change:**

```bash
make format.go
make lint
make check.build.app
make check.format.go
make check.generated.artifacts
```

**Frontend-only change:**

```bash
make format.js
make check.format.js
make check.fast.security
make check.lint.ui
make check.build.ui
make check.generated.artifacts
```

**Proto change:**

```bash
make pb.gen
make check.proto.field.numbers
make check.generated.artifacts
```

**Models change (add to Go batch):**

```bash
make check.models.tx.debt
```

**gRPC actions change (add to Go batch):**

```bash
make check.grpc.actions.status
```
