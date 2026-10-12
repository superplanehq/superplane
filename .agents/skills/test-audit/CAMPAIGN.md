# Test-pruning campaign

Campaign mode prunes one subsystem's whole test surface in one PR: a Go
package such as `pkg/workers`, a UI mapper such as
`web_src/src/pages/app/mappers/github`, or an integrated runner area such as
`pkg/runners/protocol`. The value bar, retention bar, candidate evidence, and
validation in [SKILL.md](SKILL.md) apply to every lane. This file adds the order
of work and the lessons of a full campaign.
Each step ends on its completion criterion; do not start the next step early.

## 1. Baseline

Record the subsystem's test and support line counts and every test file's
pass/fail state at a pinned `main` SHA. Keep baseline failures in their own
list: in a workers, mapper, or runner protocol campaign, treat those failures as
possible product bugs, not stale tests, until proven otherwise.

Done when every in-scope test file has a recorded baseline result.

## 2. Lanes and inventory

Split the surface into **lanes** along production owner boundaries, not file
prefixes. For `pkg/workers` these might be routing, retention, cleanup, and
stream handling. For a UI mapper they might be status, payload, empty state,
and error mapping. For `pkg/runners` they might be registration, protocol,
execution, and log upload. Include the subsystem's cases at shared core
boundaries and its end-to-end and live-proof harness tests.

Done when every test file and QA scenario the subsystem owns belongs to exactly
one lane.

## 3. Read-only ledger per lane

Give each lane to its own read-only agent. The agent reads every assigned test
in full, including parameter tables. It also reads the production owners and
their entry points, callers, history, and CI routing. Each test declaration
goes into a written **ledger** with one mark. A table-driven case or
parameterized test is one declaration unless its rows need different marks;
then mark each row.

- `R`: retain, naming the contract and the bug it catches; a retained test that
  only moves to a better-named file stays `R` with the move noted;
- `F`: retain the contract but repair the assertion, such as a vacuous negative
  that passes when only one of several items is missing;
- `C`: consolidate, naming the owner that absorbs the assertion first: a sibling
  table case, a stronger boundary suite, or the shared owner in another package;
- `D`: delete, naming the proof that remains, or why no contract exists.

Judge a test by its assertions, not its name. A mapper test named for hiding a
disconnected state may only assert the badge was not removed.

Done when every declaration in the lane has a mark and an evidence line.

## 4. Layer plan per lane

Treat the per-test ledger as input, not as the edit list. A second read-only
pass, starting from the ledger, looks for the redundant **layer**. In a
workers campaign, several suites may replay the same router through one mocked
collaborator, around stronger gRPC or queue-fixture suites. Name the
**keeper** suite for each contract. Prefer the real transport boundary with a
fake network over a mocked collaborator. Correct any ledger errors this pass
finds.

Done when each lane plan names its retired files, its keeper per contract, the
assertions to carry into keepers, and the test-only production seams unlocked.

## 5. Cutover

Edit lane by lane. Serialize changes to shared harnesses and support files
through one owner. With each lane, remove the test-only production seams it
unlocks: injection parameters, getters, reset exports, and indirection layers.
Keep Make targets and package paths accurate when suites move. Put durable
test-ownership rules in the area's `AGENTS.md` when one exists, drawn from
mistakes this campaign actually found.

Validate each lane with the smallest owner suite from [SKILL.md](SKILL.md):
`make test PKG_TEST_PACKAGES=...` for Go, `make check.test.ui FILES=...` for
UI, and `E2E_TEST_PACKAGES=... make test.e2e` (or `make test.e2e.single
FILE=... LINE=...`) for end-to-end. Then format, run `git diff --check`, and run
the matching lint and build checks from
[pre-pipeline-review](../pre-pipeline-review/SKILL.md).

Done when every lane plan is applied and each lane's keepers pass.

## 6. Preservation review

Before claiming completion, have independent reviewers compare deleted
coverage against the keepers, one reviewer per boundary group. They look for
contracts that lost their only proof. They also look for new assertions that
cannot fail, such as a rejection row the production code never reaches.

For each restored contract, make one deliberate **mutation** of the production
owner and confirm the keeper goes red. Then restore the source byte for byte.

Done when every reported gap is restored or rejected with source evidence, and
every restored contract has a caught mutation.

## 7. Product defects

A baseline failure that survives into a keeper is a bug report. Fix it at its
owner as a separate commit, and prove it through the real user flow, with a
**control** run that reverts the fix and shows the old behavior. Record
unrelated product discrepancies you find as follow-ups instead of fixing them
in the campaign.

Done when each repaired defect has a failing control and a passing candidate
on the same harness.

## 8. Reconcile and hand off

Campaigns outlive many `main` commits. Merge `main` rather than rebasing a
long, many-commit campaign. When `main` modified a file the campaign
deleted, keep the deletion. Port the new contract into the keeper instead, and
confirm every new regression `main` added still has a home. Rerun the whole
subsystem suite with the Make targets in [SKILL.md](SKILL.md) and repeat live
proof on the merged head.

Expect review tooling to see a truncated file list on a diff this large.
Record maintainer decisions in the PR evidence rather than editing gates.

Hand off with the [SKILL.md](SKILL.md) report, plus:

- baseline and final test/support line counts, with production counted separately;
- lanes, retired layers, and keepers;
- preservation gaps found and their mutations;
- product defects with control and candidate proof.
