# Changes Log

Running record of changes made while executing [PLAN.md](./PLAN.md).
Append a new entry at the bottom of each phase as work progresses.
Each entry should answer: *what changed, why, and where to verify*.

---

## Phase 1 — Fix transaction boundaries in invitation flow

Status: **skipped (deferred)**

### Note

Phase 1 is deferred. The underlying issue is real and the
self-acknowledged TODOs in
[pkg/grpc/actions/organizations/create_invitation.go:96](../pkg/grpc/actions/organizations/create_invitation.go#L96)
and [:170](../pkg/grpc/actions/organizations/create_invitation.go#L170)
remain valid follow-up work — but the right fix depends on a
non-trivial spike to determine whether Casbin's gorm-adapter can
participate in a caller-supplied `*gorm.DB` transaction (Approach A
in [PLAN.md](./PLAN.md)) or whether a saga / compensating-action
restructure is preferable (Approach B). That investigation is
interesting and worth doing, but it is **out of scope for the
current improvement task**, which is intentionally bounded.

Worth flagging for a future session:

- The orphan-role-grant hazard is latent: a rollback after
  `AssignRole` succeeds leaves a stranded `casbin_rule` row.
  Severity is low in practice (the failure window is small) but
  non-zero.
- The same root pattern affects
  [pkg/grpc/actions/organizations/remove_user.go:32](../pkg/grpc/actions/organizations/remove_user.go#L32)
  (also TODO-marked). A single follow-up PR could address all three
  call sites once Approach A vs B is decided.
- Recommended next step when revisiting: run the 1-hour spike
  described in [PLAN.md §Phase 1 → Implementation steps → 1](./PLAN.md),
  then choose A or B and proceed.

---

## Phase 2 — Outbound HTTP retry pattern (Slack)

Status: **complete (locally verified, not yet on a PR)**

### Files touched

New:
- [pkg/integrations/httpx/retry.go](../pkg/integrations/httpx/retry.go) — retry helper (`Do`, `Config`, `DefaultConfig`, `DefaultRetryOn`)
- [pkg/integrations/httpx/retry_test.go](../pkg/integrations/httpx/retry_test.go) — unit tests

Modified:
- [pkg/integrations/slack/client.go](../pkg/integrations/slack/client.go) — package-level `slackHTTPClient` (`Timeout: 30s`, nil Transport so it falls back to `http.DefaultTransport` and existing tests keep working) + package-level `slackRetryConfig`. `execRequest` now routes through `httpx.Do` and takes `body []byte` instead of `io.Reader` so the body can be replayed across retries
- [pkg/integrations/slack/test_helpers_test.go](../pkg/integrations/slack/test_helpers_test.go) — added `withFastRetries(t, attempts)` helper for sub-second retry tests
- [pkg/integrations/slack/send_text_message_test.go](../pkg/integrations/slack/send_text_message_test.go) — new subtest `transient 503 -> retried and succeeds`

### Summary

Slack outbound API calls now survive transient `5xx`/`429`/network
errors. The previous `&http.Client{}` constructed per request (no
timeout, no connection reuse) is replaced with a package-level
shared client that has a 30-second timeout and goes through a new
`httpx.Do` retry helper (3 attempts default, exponential backoff
with full jitter, honors `Retry-After`, respects context
cancellation). The helper is dependency-free (no go.mod additions)
and is intentionally placed at `pkg/integrations/httpx/` so other
integrations can adopt it incrementally in follow-up PRs.

Note (not done in this PR): `execRequest` uses
`context.Background()` because the calling component contexts
(`SetupContext`, `ExecutionContext`) do not currently expose a
`context.Context`. The integration framework already has an
`HTTPContext` interface at [pkg/core/component.go:55](../pkg/core/component.go#L55)
with a comment indicating components "should always use this
context instead of net/http directly" — a future cleanup could
migrate Slack (and other integrations) to use that abstraction
directly.

### Tests added

- `pkg/integrations/httpx/retry_test.go` — 9 top-level tests:
  - `Test__Do__SucceedsAfterTransientFailures` — 503 → 503 → 200 succeeds on 3rd attempt
  - `Test__Do__ReturnsLastResponseWhenAllAttemptsFail` — exhausted retries return final response
  - `Test__Do__DoesNotRetryOn4xx` — 400 returns immediately (1 attempt)
  - `Test__Do__HonorsRetryAfterHeader` — `Retry-After: 1` produces ≥900ms gap
  - `Test__Do__AbortsOnContextCancellation` — context cancel mid-backoff returns `ctx.Err()`
  - `Test__Do__ReplaysBodyOnRetry` — POST body received identically on each attempt
  - `Test__Do__RetriesOnNetworkError` — transport-level errors are retried
  - `Test__Do__RejectsBodyWithoutGetBody` — guards a foot-gun
  - `Test__ParseRetryAfter` (5 sub-cases), `Test__BackoffDelay__BoundsAreRespected`
- `pkg/integrations/slack/send_text_message_test.go` —
  `Test__SendTextMessage__Execute/transient_503_->_retried_and_succeeds`
  asserts 3 attempts and identical request body across them

### Validation evidence

- `gofmt -s -l pkg/integrations/httpx/ pkg/integrations/slack/` — pass (no output)
- `go vet ./pkg/integrations/httpx/ ./pkg/integrations/slack/` — pass
- `go build ./pkg/integrations/...` — pass
- `go test ./pkg/integrations/httpx/ -count=1` — pass (9/9 tests, 1.3s)
- `go test ./pkg/integrations/slack/ -count=1` — pass (all existing + new retry test)
- `make lint` and `make check.build.app` not run locally (require docker compose dev stack); will run in CI on PR

### Risk note

This change introduces a behavior shift: a flaky Slack endpoint
that previously failed fast now blocks for up to ~3× the backoff
ceiling (~5s × 3 + jitter, capped) before surfacing the error.
This is intentional and desirable, but worth noting for on-call:
outbound Slack failures will now appear as slower errors rather
than instant ones. The 30-second client-level timeout is the hard
cap on any single attempt.

### PR

_(not yet opened)_

---

## Phase 3 — Webhook idempotency

Status: **not started**

### Files touched

_(populate as work proceeds)_

### Summary

_(one paragraph once shipped)_

### Tests added

- _(test name + path)_

### Validation evidence

- `make lint` — _(pass/fail)_
- `make check.build.app` — _(pass/fail)_
- `make db.migrate DB_NAME=superplane_test` — _(pass/fail)_
- `make test PKG_TEST_PACKAGES=./pkg/models` — _(pass/fail)_
- `make test PKG_TEST_PACKAGES=./pkg/public` — _(pass/fail)_
- `make test PKG_TEST_PACKAGES=./pkg/integrations/aws/sns` — _(pass/fail)_
- `make test PKG_TEST_PACKAGES=./pkg/workers` — _(pass/fail)_

### PR

_(link)_
