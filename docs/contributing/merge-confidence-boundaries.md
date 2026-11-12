# Merge Confidence boundaries

Merge Confidence owns its execution data in `pkg/factories/mergeconfidence`.
Shared workers consume prepared values. They do not recognize check prompts,
assessment nodes, or report contents. There is no automation callback registry.

## Admission and identity

GitHub and Bitbucket PR triggers use the factory PR producer. Checks startup
uses the same producer. Merge Confidence selects the Bitbucket diff identity
and rejects redundant or delayed updates. Models retain transaction locks,
event persistence, and generic payload matching for legacy events without keys.
Description edits do not start another assessment. Retargeting and explicit
reruns retain their existing behavior.

Factory column-app admission links the run to the stored PR before agents can
start. The stored PR selects the work order. Owner and subscription model checks
remain mandatory. Permanent rejections retain their task note and failed run.
Transient failures retain transaction retry behavior.

## Runner credentials

The feature owner prepares runner credentials with the caller's transaction.
Subscription runs resolve the work order from persisted run authority. The root
PR event supplies the repository. Saved prompts supply enabled check IDs and
custom labels. Legacy combined prompts remain supported.

Prepared environment values are trusted internal context data. Component
configuration cannot supply these values directly. Shared runner preparation
removes internal markers and applies the prepared credentials. Ordinary agents
retain their planning and task paths, including Additional instructions.

Signed claims, tool permissions, legacy token handling, and active execution
authorization remain runner contracts. Moving preparation does not relax them.

## Completion and display

The feature owner prepares separate passed and failed completion values. It uses
the run's saved canvas version, run-scoped checks, and root-event commit. Shared
emission persists those ordinary values and records configured channel errors.
A failed assessment remains failed after successful downstream publication.
Finished and cancelling executions keep their existing guards.

Each agent poll uses one saved task-status snapshot. Report preparation runs
only when that snapshot is terminal, before the handler emits its final output.
Running tasks and lookup failures use the ordinary execution context. The poll
handler keeps its scheduling and retry behavior and consumes the same snapshot.
A task that finishes after the snapshot waits until the next poll. This keeps
the final reported check and terminal status consistent. Cancellation, callbacks,
and other hooks retain their existing context preparation.

Publication and completion broadcast remain explicit canvas steps. Review
request consumes the trusted broadcast through its filtered On Broadcast
trigger. Original-report retries and current-commit checks remain in that graph.

The Merge Confidence UI prepares display steps after selecting the live or
historical graph. Shared rendering receives ordinary step data. Other agent
runs retain their saved step names.

This change requires no new service, environment variable, permission, or
deployment configuration. Later checks and Review request settings must preserve
these identity and authorization boundaries.
