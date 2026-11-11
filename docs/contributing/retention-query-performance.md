# Retention candidate selection performance

## Local comparison methodology

The local comparison used temporary tables and indexes in one PostgreSQL session.
It did not change application data or production indexes.

The fixture contained 100 organizations, 1,000 workflows, 200,000 runs,
200,000 executions, 200,000 requests, and 2,000 queue items.
Retention windows included null, zero, 7, 30, and 90 days.
Runs included unfinished runs, active executions, queue items, and pending requests.
Each run had a 512-character input payload.
The fixed UTC reference time was `2026-11-01 12:00:00`.

The comparison covered the complete selection with all three safety checks,
oldest-first ordering, and a 1,000-row limit.
It checked equality of all eligible IDs before applying the limit.
It ran `ANALYZE` and seven measurements for each variant and plan mode.
It reported median execution time and buffer work, plus one complete JSON plan
for each variant and mode.

## Local observation: November 11, 2026

PostgreSQL 17.5, Linux amd64, `shared_buffers=128MB`, `work_mem=4MB`,
and `random_page_cost=4`. No concurrent application workload ran during the final comparison.
Temporary tables use local buffers and cannot use parallel scans.
These results compare access paths. They do not represent production statement latency.

Seven samples per cell, including the first execution:

| Dataset and plan | Baseline median | Index only | Index and predicate rewrite | Final selection |
| --- | ---: | ---: | ---: | ---: |
| Expired batch, custom | 128.531 ms | 9.004 ms | 8.721 ms | 8.741 ms |
| Expired batch, generic | 569.782 ms | 543.287 ms | 544.998 ms | 8.751 ms |
| Recent empty, custom | 63.367 ms | 171.749 ms | 171.016 ms | 0.035 ms |
| Recent empty, generic | 64.243 ms | 63.122 ms | 64.039 ms | 0.024 ms |

The expired fixture has 72,850 eligible runs. Selection returns 1,000 rows.
The recent fixture has no eligible runs.

| Dataset and plan | Baseline local hits / reads | Final local hits / reads | Baseline / final temporary writes |
| --- | ---: | ---: | ---: |
| Expired batch, custom | 7 / 22,785 | 17,510 / 3,738 | 0 / 0 |
| Expired batch, generic | 487,213 / 263,857 | 17,521 / 3,736 | 6,791 / 0 |
| Recent empty, custom | 5 / 16,680 | 4 / 0 | 0 / 0 |
| Recent empty, generic | 0 / 16,685 | 4 / 0 | 0 / 0 |

The baseline custom plan scans runs, joins organizations, and sorts candidates.
The generic baseline does many execution and request probes and spills its sort.
The final plans scan the partial completion-time index in order and stop after 1,000 eligible rows.
They calculate the shortest enabled retention window once as an InitPlan.
This cutoff prevents scanning recent runs that cannot expire under any organization policy.
The original organization-specific expiration predicate remains the authoritative check.

A parameterized finished-state predicate prevents generic plans from proving the partial index condition.
The final query uses the fixed finished-state literal; reference times and active-work states remain parameters.
The organization-dependent predicate rewrite gave no useful additional improvement.
The index alone regressed empty-result calls, so the final query adds the minimum-window cutoff.
No included columns or workflow-leading index were needed for these plans.

## Rollout and production acceptance

Apply the standalone concurrent index migration before deploying the optimized query.
Review index build duration and locking against the deployed table size.
Many blocked or long-retention runs can still require substantial scans.
A short retention window on an organization with no runs can weaken the global cutoff.

Production acceptance remains **pending**. This runner has no affected production query measurements.
A maintainer must collect at least 100 naturally occurring worker selection calls under normal load.
Measure database statement duration, including result production, for the complete 1,000-candidate statement.
Exclude lock and deletion statements. Include populated and empty calls when they occur.
Record the UTC observation window, sample count, workload conditions, median, p95, and latency distribution.
The p95 must be strictly below 500 milliseconds.
Do not use these local `EXPLAIN` timings or artificial warm-cache calls to claim production acceptance.
