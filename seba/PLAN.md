# Improvement Plan — Three Phases

Owner: Seba · Created: 2026-04-28 · Repo: superplanehq/superplane

This plan executes three independent improvements identified in
[DIAGNOSTICS.md](./DIAGNOSTICS.md). Each phase is self-contained,
ships behind its own PR, and can be paused/abandoned without
blocking the others. As phases land, append entries to
[CHANGES.md](./CHANGES.md).

---

## Phase 1 — Fix transaction boundaries in invitation flow

### Goal

Eliminate the orphan-role-grant hazard in
[pkg/grpc/actions/organizations/create_invitation.go](../pkg/grpc/actions/organizations/create_invitation.go).
After the fix, a rollback of the user/invitation transaction must
not leave a stranded entry in `casbin_rule`.

### Scope

- [pkg/authorization/interface.go](../pkg/authorization/interface.go) — add new method to interface
- [pkg/authorization/service.go](../pkg/authorization/service.go) — implement new method
- [pkg/grpc/actions/organizations/create_invitation.go](../pkg/grpc/actions/organizations/create_invitation.go) — both call sites at line 98 and line 172
- [pkg/grpc/actions/organizations/create_invitation_test.go](../pkg/grpc/actions/organizations/create_invitation_test.go) — extend with rollback test

Out of scope: `remove_user.go:32` (similar TODO, but a separate PR
keeps blast radius small).

### Design decision

Two viable approaches. The plan picks **Approach A** as preferred,
with **Approach B** as a fallback if Approach A turns out to fight
Casbin's adapter model.

#### Approach A — Transactional auth method (preferred)

Add `AssignRoleInTransaction(tx *gorm.DB, userID, role, domainID, domainType string) error`
to the `Authorization` interface. The implementation builds a
short-lived Casbin enforcer wired to the passed `tx` (via
`gormadapter.NewAdapterByDB(tx)` or by reusing the existing
enforcer with a tx-bound adapter swap), so policy mutations
participate in the caller's transaction. Caller in
`create_invitation.go` switches from `AssignRole(...)` to
`AssignRoleInTransaction(tx, ...)`.

Pros: directly correct, matches the AGENTS.md
`*InTransaction` convention used throughout `pkg/models`.

Risks: Casbin's gorm-adapter has nuances around rebuilding policy
state and concurrent enforcer access. Spike before committing.

#### Approach B — Saga / compensating action (fallback)

Keep `AssignRole` as-is. Restructure the invitation flow so the
GORM transaction commits *first*; only then call `AssignRole`. If
`AssignRole` fails, execute a compensating delete of the user and
invitation rows.

Pros: no Casbin changes; smaller PR.

Cons: failure window between commit and AssignRole leaves a brief
inconsistent state visible to readers; compensating delete needs
careful error handling so it does not itself fail silently.

### Implementation steps

1. **Spike (timeboxed, 1 hour):** in a throwaway branch, prove that
   Casbin's gorm-adapter accepts a `*gorm.DB` that is mid-transaction
   and that policy mutations made through it participate in the
   caller's tx (verify by triggering a rollback and checking
   `casbin_rule` is unchanged).
2. If spike succeeds → proceed with **Approach A**. If not →
   **Approach B**.
3. **Approach A path:**
   a. Add `AssignRoleInTransaction(tx *gorm.DB, ...)` to
      [pkg/authorization/interface.go](../pkg/authorization/interface.go).
   b. Implement it in [pkg/authorization/service.go](../pkg/authorization/service.go)
      next to `AssignRole`. Have `AssignRole` delegate via
      `return a.AssignRoleInTransaction(database.Conn(), ...)`.
   c. Update the closure-style block at
      [create_invitation.go:65-99](../pkg/grpc/actions/organizations/create_invitation.go#L65)
      to call `authService.AssignRoleInTransaction(tx, ...)`.
   d. Refactor `handleNewUser` at
      [create_invitation.go:110-186](../pkg/grpc/actions/organizations/create_invitation.go#L110)
      to use the closure-based
      `database.Conn().Transaction(func(tx) error {...})`
      pattern instead of the manual `tx.Begin()/Rollback()/Commit()`
      sequence, and call `AssignRoleInTransaction(tx, ...)` from
      inside.
   e. Remove both `// TODO: this is not using the transaction
      properly` comments.
4. **Approach B path:** restructure both flows to commit the GORM
   transaction first, then call `AssignRole`, with a
   compensating-delete `defer` that triggers only if `AssignRole`
   returns an error.

### Tests / validation

- Existing tests must continue passing: `make test PKG_TEST_PACKAGES=./pkg/grpc/actions/organizations`
  and `make test PKG_TEST_PACKAGES=./pkg/authorization`.
- New test in
  [create_invitation_test.go](../pkg/grpc/actions/organizations/create_invitation_test.go):
  *"role grant rolls back when invitation creation fails"*. Use a
  test hook or constraint violation to force the user-creation
  step to fail mid-transaction; assert that no row exists in
  `casbin_rule` for the would-be user. Mirror this for both the
  new-account and existing-account flows.
- For Approach A specifically: add a test in
  [pkg/authorization/service_test.go](../pkg/authorization/service_test.go)
  that calls `AssignRoleInTransaction` inside a tx that is
  explicitly rolled back, then verifies the policy is gone.

### Validation commands

```
make format.go
make lint
make check.build.app
make test PKG_TEST_PACKAGES=./pkg/authorization
make test PKG_TEST_PACKAGES=./pkg/grpc/actions/organizations
```

### Acceptance criteria

- [ ] Both `// TODO: this is not using the transaction properly`
      comments are removed and the code behind them is correct
- [ ] New negative test demonstrates that a forced rollback leaves
      no role row in `casbin_rule`
- [ ] No new lint warnings; existing test suites green
- [ ] PR description references this plan and explains which
      approach was taken (and why, if Approach B)

### Risks / rollback

The change touches the `Authorization` interface (Approach A). If
issues surface in production, revert is a single-PR revert. No
schema changes, so no database rollback needed.

---

## Phase 2 — Outbound HTTP retry pattern (Slack as proof of concept)

### Goal

Establish a reusable, dependency-free retry helper for outbound
integration HTTP calls and apply it to Slack. Same change fixes
the no-timeout `&http.Client{}` hazard.

### Scope

- New file: `pkg/integrations/httpx/retry.go` (small package; the
  name avoids stomping on `pkg/lib/`-style utilities)
- New file: `pkg/integrations/httpx/retry_test.go`
- Modified: [pkg/integrations/slack/client.go](../pkg/integrations/slack/client.go)
- Modified: [pkg/integrations/slack/slack_test.go](../pkg/integrations/slack/slack_test.go)
  — extend with retry-behavior tests

Out of scope: rolling the helper out to other integrations. That is
deliberately a separate ramp-out PR per integration once the
pattern is validated.

### Design decision

Build a tiny in-house helper — no new go.mod dependency. Roughly
50 lines: exponential backoff with jitter, capped attempts,
configurable retry-on predicate, respects `Retry-After` header
when present.

Rationale: the project keeps a small dependency surface, and the
retry contract here is simple enough that a third-party library
would obscure rather than help. If we later need richer behavior
(circuit breakers, adaptive timeouts), reach for `sony/gobreaker`
at that point.

### Implementation steps

1. Create `pkg/integrations/httpx/retry.go` exposing:
   ```go
   type Config struct {
       MaxAttempts   int           // default 3
       BaseDelay     time.Duration // default 200ms
       MaxDelay      time.Duration // default 5s
       RetryOn       func(*http.Response, error) bool // default: 5xx, 429, net errors
   }

   func Do(ctx context.Context, client *http.Client, req *http.Request, cfg Config) (*http.Response, error)
   ```
   - Honor `ctx` cancellation between attempts
   - Honor `Retry-After` header on 429/503 (cap by `MaxDelay`)
   - Add full jitter to the backoff
   - Read and discard body on retried responses to free the
     connection
2. In [pkg/integrations/slack/client.go](../pkg/integrations/slack/client.go):
   - Replace per-request `&http.Client{}` with a package-level
     `var slackHTTPClient = &http.Client{Timeout: 30 * time.Second}`
   - Route `execRequest` through `httpx.Do(ctx, slackHTTPClient, req, httpx.Config{})`
   - Thread `context.Context` through `execRequest` if it is not
     already (verify; today the function signature does not take
     `ctx`)
3. Update Slack tests to drive the retry path with `httptest.Server`.

### Tests / validation

- Unit tests in `pkg/integrations/httpx/retry_test.go`:
  - 503 → 503 → 200 ⇒ succeeds on third attempt
  - All attempts return 500 ⇒ returns last response with no error
    OR returns aggregated error (decide and document)
  - 429 with `Retry-After: 2` ⇒ honors header (use a fake clock or
    assert duration ≥ 2s with tolerance)
  - `ctx.Done()` mid-backoff ⇒ aborts with `ctx.Err()`
  - Network error (close-on-accept) ⇒ retries, then fails
- Slack integration test in
  [pkg/integrations/slack/slack_test.go](../pkg/integrations/slack/slack_test.go):
  - `httptest.Server` returns 503 then 200 ⇒ `SendMessage` succeeds
  - Assert exactly 2 requests reached the test server

### Validation commands

```
make format.go
make lint
make check.build.app
make test PKG_TEST_PACKAGES=./pkg/integrations/httpx
make test PKG_TEST_PACKAGES=./pkg/integrations/slack
```

### Acceptance criteria

- [ ] Slack outbound calls survive a transient 503
- [ ] Slack `http.Client` has a configured timeout (no more naked
      `&http.Client{}`)
- [ ] `httpx.Do` ships with ≥4 unit tests covering retry,
      `Retry-After`, context cancellation, and terminal failure
- [ ] No new go.mod dependency
- [ ] PR description names the next 1-2 integrations expected to
      adopt the helper, but does not migrate them in this PR

### Risks / rollback

Behavior change: a flaky Slack endpoint that previously failed fast
will now block for up to ~7s (3 attempts × backoff). This is
desirable for reliability but worth calling out so on-call sees
slower outbound failures rather than instant ones. Single-PR revert
removes the helper.

---

## Phase 3 — Webhook idempotency

### Goal

Prevent duplicate processing when a webhook is delivered twice
(provider replay, network retry, etc.). Establish the framework
seam and roll out to two providers in this PR: AWS SNS (uses
`MessageId` in the body) and GitHub (uses `X-GitHub-Delivery`
header).

### Scope

- New migration: `db/migrations/{ts}_create-webhook-deliveries.up.sql`
  (and empty `.down.sql`)
- New file: `pkg/models/webhook_delivery.go`
- Modified: [pkg/core/trigger.go](../pkg/core/trigger.go) — add
  optional `DedupKey` extraction
- Modified: [pkg/public/server.go](../pkg/public/server.go) lines
  ~1115 and ~1150 — central dispatch checks idempotency before
  invoking `HandleWebhook`
- Modified: [pkg/integrations/aws/sns/on_topic_message.go](../pkg/integrations/aws/sns/on_topic_message.go)
  — implement dedup-key extraction
- Modified: GitHub trigger files — implement dedup-key extraction
- New worker: `pkg/workers/webhook_delivery_cleanup_worker.go`
  prunes expired rows (registered in `cmd/server/main.go` per
  AGENTS.md guidance)
- Tests for each modified file

### Design decision

**Where dedup happens:** centrally, in the webhook dispatch in
[pkg/public/server.go:1115](../pkg/public/server.go#L1115).
Per-trigger logic only *extracts* the dedup key from the request;
the framework handles persistence and the duplicate check.

**How dedup keys are surfaced by triggers:** add an optional
interface method `DedupKey(ctx WebhookRequestContext) (string, bool)`.
A trigger that does not implement it (or returns `false`) opts out
of idempotency — preserving today's behavior for integrations not
yet migrated.

**Storage:** `webhook_deliveries(id, node_id, dedup_key, received_at, expires_at)`
with `UNIQUE(node_id, dedup_key)`. Lookup by `INSERT ... ON CONFLICT
DO NOTHING` — the row count tells us whether this is a new delivery.
TTL: 7 days (rotates by cleanup worker; long enough to catch every
realistic provider retry window).

### Implementation steps

1. **Migration:**
   `make db.migration.create NAME=create-webhook-deliveries`,
   then write the table DDL with the unique index. Run
   `make db.migrate DB_NAME=superplane_dev` and
   `make db.migrate DB_NAME=superplane_test`.
2. **Model:** `pkg/models/webhook_delivery.go` with
   `RecordWebhookDeliveryInTransaction(tx, nodeID, dedupKey, ttl) (alreadySeen bool, err error)`
   plus a non-tx wrapper, following the project convention.
3. **Trigger interface:** in
   [pkg/core/trigger.go](../pkg/core/trigger.go), add a separate
   optional interface (do not extend `Trigger` itself, so the
   change is non-breaking):
   ```go
   type DedupKeyExtractor interface {
       DedupKey(ctx WebhookRequestContext) (string, bool)
   }
   ```
4. **Dispatcher:** in
   [pkg/public/server.go](../pkg/public/server.go), wrap both
   `HandleWebhook` call sites: type-assert the trigger/action to
   `DedupKeyExtractor`; if it implements it and returns a key,
   call `RecordWebhookDelivery`; if `alreadySeen`, return
   `200 OK` with an empty body and skip dispatch.
5. **Roll out to two providers:**
   - SNS: `DedupKey` returns the body's `MessageId`.
   - GitHub: `DedupKey` returns `X-GitHub-Delivery` header.
6. **Cleanup worker:** new file in `pkg/workers/`, following the
   pattern of [pkg/workers/integration_cleanup_worker.go](../pkg/workers/integration_cleanup_worker.go).
   Register it in `cmd/server/main.go` per
   [AGENTS.md](../AGENTS.md) §Build, Test, and Development Commands.
7. **Docker compose:** if the worker introduces env vars (cleanup
   interval, retention days), add them to the compose files per
   AGENTS.md.

### Tests / validation

- Unit tests in `pkg/models/webhook_delivery_test.go`: insert,
  duplicate-insert (assert `alreadySeen=true`), expiry filtering.
- Dispatcher test in `pkg/public/server_test.go` (or a new
  integration test): post the same SNS payload twice, assert only
  one event is emitted downstream (use the existing test
  `EventContext` mock pattern).
- SNS-specific test in
  [pkg/integrations/aws/sns/on_topic_message_test.go](../pkg/integrations/aws/sns/on_topic_message_test.go):
  exercise `DedupKey` with and without a `MessageId` field.
- Cleanup-worker test: insert expired rows, run worker tick, assert
  rows gone.

### Validation commands

```
make format.go
make lint
make check.build.app
make db.migrate DB_NAME=superplane_test
make test PKG_TEST_PACKAGES=./pkg/models
make test PKG_TEST_PACKAGES=./pkg/public
make test PKG_TEST_PACKAGES=./pkg/integrations/aws/sns
make test PKG_TEST_PACKAGES=./pkg/workers
```

### Acceptance criteria

- [ ] Replaying an SNS or GitHub webhook delivery does not produce
      duplicate downstream events
- [ ] Triggers that do not implement `DedupKeyExtractor` continue
      to work exactly as before (no behavior change)
- [ ] `webhook_deliveries` table exists with `UNIQUE(node_id, dedup_key)`
- [ ] Cleanup worker runs in the dev/test docker compose stacks
- [ ] PR description lists the next 2-3 providers expected to adopt
      `DedupKeyExtractor` (Slack, PagerDuty, Stripe-style, etc.)

### Risks / rollback

The migration adds a table; rollback is non-destructive (drop
table). The dispatcher change is backward-compatible because the
interface is opt-in. Worst case: an unbounded growth of the table
if the cleanup worker is not started — mitigated by the worker
ticking on a short interval and the `expires_at` filter.

---

## Sequencing

Phases are independent and can be worked in any order, but the
suggested order is **1 → 2 → 3** because:
- Phase 1 is a correctness fix in a small surface area; safest to
  ship first.
- Phase 2 introduces a reusable building block (`httpx`) that
  Phase 3's cleanup worker may want to use for its own outbound
  calls (none today, but possible later).
- Phase 3 is the largest of the three (migration + worker + new
  interface) and benefits from going last when the rhythm is set.

## Out of scope (deliberately)

- Frontend findings (key={index}, file-wide eslint disables, brand
  casing) — separate, smaller PRs.
- Logging consistency in PagerDuty webhook — separate PR, mechanical.
- `interface{}` → `any` mechanical sweep — low value alone, only
  worthwhile if bundled with another change in the same file.
