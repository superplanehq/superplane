# Codebase health

## Overview

This PRD specifies **Codebase Health**: a factory-level surface that
aggregates verification findings over time into a **health score**,
**streaks**, **recurring patterns**, and **achievements**, plus a structured
**run summary report** per verification run. The goal is sustained engagement
with code quality: teams see progress, not only failures.

This document is a specification. It does not ship backend or application
code. The companion Storybook designs
(`web_src/src/pages/factories/health/`) show the intended UI. It consumes the
findings defined in [line-verification.md](line-verification.md) and the
aggregation defined in
[verification-suggestions.md](verification-suggestions.md).

## Problem Statement

Verification produces a stream of pass/fail results and findings. Streams are
easy to ignore. Without an aggregate view, a team cannot answer basic
questions: is quality improving, which problems repeat, which repositories
need attention, and did last month's cleanup hold.

Progress that is visible gets sustained. A score with a trend, a streak that
a failing run would break, and named recurring problems with a clear path to
zero give teams a reason to keep verification green — and give leads a
truthful summary instead of anecdotes.

SuperPlane already renders exactly the right primitives: console scorecard
panels with value, change chip, sparkline, and target
([console-and-widgets.md](console-and-widgets.md)). Codebase Health reuses
this visual language at the factory level.

## Goals

1. Compute a **health score** per factory (and per repository where the
   factory spans several) from severity-weighted findings over time, with
   trend and target.
2. Track **streaks**: consecutive work orders and consecutive days with zero
   new blocking findings.
3. Name **recurring patterns**: repeated finding groups presented as cards
   with plain descriptions, top offender files, common cause, and standard
   remediation.
4. Award **achievements** for meaningful milestones, with descriptive names
   that state the accomplishment directly.
5. Emit a **run summary report** after each verification run: findings
   detected, fixed, and remaining, by severity; postable to Slack through the
   existing integration.

## Non-Goals

- No individual-person scoring in v1. Scores, streaks, and achievements
  attach to factories, lines, and repositories, not to people. The optional
  team leaderboard compares factories or lines only.
- No paging or alerting; the run summary report is informational.
- No external gamification integrations.
- No changes to how findings are produced; this PRD is read-side only.

## Concepts

### Health score

A number from 0 to 100 per factory, recomputed per verification run:

- Start from 100; subtract weighted open findings (`high` > `medium` >
  `low`), normalized by the volume of verified work so large factories are
  comparable to small ones.
- Blocking findings weigh more than advisory findings at equal severity.
- The score carries a trend (versus the previous period) and an optional
  target the team sets.

The exact formula is an implementation decision; the contract is: fixing
findings raises the score, new findings lower it, and severity orders the
impact.

### Streaks

Two counters per factory:

- **Work order streak** — consecutive work orders whose verification passed
  with zero new blocking findings.
- **Daily streak** — consecutive days with at least one verification run and
  zero new blocking findings.

A qualifying failure resets the counter. Days without runs pause, not reset,
the daily streak.

### Recurring patterns

A recurring pattern is a named group of findings that repeat: same rule,
similar locations, across runs and work orders (grouping from
[verification-suggestions.md](verification-suggestions.md)). A pattern card
shows:

- A plain, descriptive name (for example "Untyped API response handling").
- What it is and why it recurs, in one or two sentences.
- Top offender files with occurrence counts.
- The standard remediation.
- An occurrence trend.

Reducing a pattern's open count to zero completes it and awards an
achievement.

### Achievements

Milestone records with descriptive names that state the fact:

- "First verification passed"
- "30 days without a type-safety finding"
- "Zero secrets findings for 90 days"
- "Recurring pattern resolved: <pattern name>"

Achievements are permanent once earned; a later regression does not remove
one, but the same achievement can be earned again after the streak rebuilds.
An optional leaderboard compares factories or lines on score and streaks;
it is off by default.

### Run summary report

A structured summary attached to the work order after each verification run:

| Field | Meaning |
| --- | --- |
| `detected` | New findings this run, by severity. |
| `fixed` | Previously open findings no longer reported, by severity. |
| `remaining` | Open findings after this run, by severity. |
| `gate` | Passed or failed, with the blocking finding count. |

The report renders on the work order timeline and can post to a Slack channel
through the existing Slack integration.

## UX walkthrough

The Storybook designs under `web_src/src/pages/factories/health/` define the
visual contract:

- **`FactoryHealthPage`** — the full Health tab: score, streaks, patterns,
  and achievements composed in the style of the existing factory Overview
  page.
- **`HealthScoreCard`** — reuses the console scorecard visual language:
  value, trend chip, sparkline, target.
- **`StreakIndicator`** — the two streaks with their current counts and best
  values.
- **`AchievementsGrid`** — earned and not-yet-earned achievements;
  not-yet-earned entries show what remains to earn them.
- **`RecurringPatternCard`** — the pattern contract above, with a link to the
  matching recurring suggestions.
- **`RunSummaryReportCard`** — the report as it renders on a work order and
  as a Slack message preview.

## Future backend surface (specification only)

Recorded for a later implementation phase; nothing here is built now.

- **Aggregation**: a worker that folds verification runs into per-factory
  daily health snapshots, streak counters, pattern groups, and achievement
  records; append-only, recomputable from findings.
- **Models**: `HealthSnapshot`, `Achievement`, and pattern aggregates in
  `pkg/models`.
- **APIs**: read-only endpoints for the Health tab; registered in
  `pkg/authorization/interceptor.go` under factory read permissions.
- **Slack**: run summary posting composes the existing Slack integration
  components; no new integration surface.

## Acceptance Criteria (for the eventual implementation)

1. The health score updates after each verification run and moves in the
   documented direction for fixed and new findings.
2. Streaks increment, pause, and reset per the rules above.
3. A pattern card appears when a finding group repeats across runs, and
   completes when its open count reaches zero.
4. Achievements are awarded exactly once per qualifying event and persist.
5. The run summary report matches the run's findings and posts to Slack when
   configured.

## Open Questions

- What period does the score trend compare against — previous week, previous
  30 days, or configurable?
- Are pattern names authored by an agent, by users, or generated from the
  rule name plus location group?
- Should the leaderboard be visible to all members or only to factory
  admins?
