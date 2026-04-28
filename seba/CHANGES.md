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

Status: **dropped — see design note below for the recommended
future implementation**

### Why this phase was dropped

Phase 3 was prototyped end-to-end with a Postgres-backed dedup
table (`webhook_deliveries`), a `DedupKeyExtractor` interface in
`pkg/core/trigger.go`, dispatcher hooks in `pkg/public/server.go`,
adoption on AWS SNS and GitHub `OnPush`, and a periodic cleanup
worker. The implementation was complete and tests passed for the
leaf packages.

After review we decided **the persistence layer is wrong for this
use case**. Webhook deliveries are *live, ephemeral* signals —
provider retry windows are minutes-to-hours, not days — and the
problem space is "have I seen this in the last few minutes?"
rather than "store this fact durably." Postgres works, but it's
the heaviest possible tool: every webhook delivery becomes a write
that has to be transactionally committed, indexed, retained, and
later pruned by a worker. For a high-fan-in webhook receiver
that's a real ops burden (table growth, vacuum pressure, an
extra worker to keep alive) for a problem that an in-memory
key/value store with native TTL solves in a single round trip.

The full prototype was reverted in this turn. Nothing under
`pkg/`, `db/migrations/`, or `docker-compose.dev.yml` retains any
trace of it.

### Why dedup still matters (do not skip it forever)

Even though Phase 3 is dropped *as implemented*, the underlying
problem is real and worth fixing in a future pass:

- **Provider retries are normal, not exceptional.** AWS SNS
  retries an unacked HTTP delivery for up to ~1 hour by default
  with no idempotency guarantees. GitHub retries on 5xx and
  declares timeouts after 10 seconds. Slack replays on
  `X-Slack-Retry-Num` after timeouts. Without dedup, a slow
  handler that 200s on the second attempt has *already processed
  the first one to completion* — the second processing is
  duplicate work the user paid for once.
- **Duplicate processing has user-visible side effects.** A
  duplicate `OnPush` event triggers a duplicate workflow run; a
  duplicate SNS notification creates duplicate downstream queue
  items; a duplicate PagerDuty webhook can re-acknowledge an
  incident. The blast radius scales with how connected the
  trigger is.
- **The current dispatcher in
  [pkg/public/server.go](../pkg/public/server.go) at the
  `executeTriggerNode` / `executeActionNode` seam is the right
  place to enforce dedup centrally** — exactly one type assertion
  away from per-integration handlers. The dropped prototype
  validated this seam works. The interface design
  (`DedupKeyExtractor` returning `(key, ok)`) is also reusable.

### Recommended future implementation: Redis-backed (or similar)

When this gets re-picked up, the right shape is:

1. **Storage: Redis (or any TTL-native KV).** A single
   `SET key value NX EX <ttl>` is the entire dedup operation.
   The reply tells you whether the key was new (`OK`) or already
   seen (`nil`). No worker, no table, no migration, no schema
   coupling. Memory cost is `~100 bytes × throughput × TTL`.
   For a TTL of 1 hour at 100 webhook/sec that's ~36MB —
   negligible.
   - Equivalent stores work just as well: KeyDB, Dragonfly, an
     in-process LRU like `hashicorp/golang-lru/v2` if the
     deployment is single-instance, or a dedicated etcd lease.
     The interface contract (SETNX + TTL) is the same.
2. **Interface seam: keep `DedupKeyExtractor` from the prototype.**
   The non-breaking design (separate optional interface, not an
   extension of `Trigger`/`Action`) is correct. Per-integration
   `DedupKey(ctx WebhookRequestContext) (string, bool)`
   implementations stay 3-5 lines each.
3. **Dispatcher hook: same place as the prototype.** Call the
   extractor, then `SETNX` against Redis with a 1-hour TTL
   (tuned per provider if needed). On a duplicate, return
   `200 OK` and skip dispatch.
4. **Failure mode: fail-open.** If Redis is unavailable, log and
   proceed to dispatch. Webhook delivery must not depend on
   dedup-store availability — at worst we revert to today's
   "process every delivery" behavior. This is the opposite of
   the Postgres design, which would block on DB unavailability
   anyway because the rest of the request path needs Postgres.
5. **Operational cost to onboard.** New `go.mod` dependency
   (`github.com/redis/go-redis/v9`), new `redis` service in
   `docker-compose.dev.yml`, new env var (`REDIS_URL`), and a
   redis service in production deploy artifacts (Helm chart +
   single-host installer). This is real but bounded — and Redis
   is also useful for other near-term features (rate limiting,
   websocket fan-out, ephemeral locks) so the cost amortizes.

### What to carry forward when re-picking this up

The prototype is not preserved in source, but the design ideas
worth keeping are:

- **Namespaced keys.** `"sns:" + MessageId`,
  `"github:" + X-GitHub-Delivery`. Prevents collisions across
  providers and makes Redis keys self-describing.
- **Scope dedup per node, not just per provider.** The same
  provider message arriving at two configured nodes is two
  separate deliveries. Compose the key as
  `"webhook:dedup:" + workflowID + ":" + nodeID + ":" + providerKey`.
- **Opt-out semantics.** `DedupKey` returning `ok=false` lets
  triggers skip dedup for malformed bodies (don't poison the
  store) and for legitimate re-delivery flows like SNS
  `SubscriptionConfirmation`.
- **Record-before-dispatch.** Trade-off intentionally chosen:
  prefer a small risk of dropping a legitimate replay over
  double-processing under handler latency.

### Files touched

None. The prototype was reverted before this entry was written.

### PR

_(not opened — phase dropped)_
