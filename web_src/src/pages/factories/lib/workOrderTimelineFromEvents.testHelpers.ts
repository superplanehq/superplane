import type { FactoriesWorkOrderEvent } from "@/api-client";

/**
 * Builds a `step.execution.*` work order event for timeline tests.
 *
 * Shared across the `workOrderTimelineFromEvents` spec files so the fixture
 * stays consistent while each spec focuses on a single behaviour area.
 */
export function stepExecutionEvent(
  type: "step.execution.created" | "step.execution.finished",
  timestamp: string,
  runState: string,
  runResult?: string,
  overrides: { stepName?: string; runId?: string } = {},
): FactoriesWorkOrderEvent {
  return {
    timestamp,
    type,
    event: {
      stepName: overrides.stepName ?? "Build",
      line: { id: "line-1", name: "CI" },
      run: { id: overrides.runId ?? "run-1", state: runState, result: runResult },
      app: { id: "app-1" },
    },
  };
}
