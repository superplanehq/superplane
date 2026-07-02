import type { CanvasesCanvasNodeExecution, CanvasesCanvasRun, SuperplaneComponentsNode } from "@/api-client";
import rawData from "./canvasRunsExample.json";

// canvasRunsExample.json is a sanitized snapshot captured from the public
// "Clean Code Assessment" canvas on app.superplane.com. It contains 25 real
// runs (11 passed, 14 failed) plus the full executions for one passed run
// and one failed run, used to drive the bottom RunNodeDetailPane. The
// GitHub PR webhook payloads in each rootEvent are trimmed to just the
// fields the github.onPullRequest trigger renderer reads.

type RawFixture = {
  canvas: {
    id: string;
    organizationId: string;
    name: string;
    nodes: SuperplaneComponentsNode[];
  };
  runs: CanvasesCanvasRun[];
  totalCount: number;
  actions: Array<{ name: string; icon: string }>;
  triggers: Array<{ name: string; icon: string }>;
  failedRunExecutions: CanvasesCanvasNodeExecution[];
  passedRunExecutions: CanvasesCanvasNodeExecution[];
  selectedFailedRunId: string;
  selectedFailedRootEventId: string;
  selectedPassedRunId: string;
  selectedPassedRootEventId: string;
};

const data = rawData as unknown as RawFixture;

export const canvasFixture = data.canvas;
export const runsFixture: CanvasesCanvasRun[] = data.runs;

export const componentIconMap: Record<string, string> = Object.fromEntries(
  [...data.actions, ...data.triggers]
    .filter((entry) => entry.name && entry.icon)
    .map((entry) => [entry.name, entry.icon]),
);

export interface RunDetailExample {
  run: CanvasesCanvasRun;
  rootEventId: string;
  nodeId: string;
  executions: CanvasesCanvasNodeExecution[];
}

function findRun(id: string): CanvasesCanvasRun {
  const run = data.runs.find((candidate) => candidate.id === id);
  if (!run) {
    throw new Error(`Fixture missing run with id ${id}`);
  }
  return run;
}

export const failedRunDetail: RunDetailExample = {
  run: findRun(data.selectedFailedRunId),
  rootEventId: data.selectedFailedRootEventId,
  // The failed run terminates on analyze-pr (RESULT_FAILED with a clear error
  // message), so it's the most informative node to feature in the bottom pane.
  nodeId: "analyze-pr",
  executions: data.failedRunExecutions,
};

export const passedRunDetail: RunDetailExample = {
  run: findRun(data.selectedPassedRunId),
  rootEventId: data.selectedPassedRootEventId,
  nodeId: "post-assessment",
  executions: data.passedRunExecutions,
};

// Live capture only contains passed / failed runs. Synthesize a running and a
// cancelled run by cloning recent passes so all four RUN_STATUS_META variants
// show up in the AllOutcomes story.
function synthesizeRun(
  template: CanvasesCanvasRun,
  overrides: Partial<CanvasesCanvasRun>,
  suffix: string,
  ageMs: number,
): CanvasesCanvasRun {
  const baseRootEvent = template.rootEvent;
  const rootEvent = baseRootEvent
    ? {
        ...baseRootEvent,
        id: `${baseRootEvent.id}-${suffix}`,
        createdAt: new Date(Date.now() - ageMs).toISOString(),
      }
    : undefined;
  return {
    ...template,
    id: `${template.id}-${suffix}`,
    createdAt: new Date(Date.now() - ageMs).toISOString(),
    rootEvent,
    ...overrides,
  };
}

const newestPassed = data.runs.find((run) => run.result === "RESULT_PASSED") ?? data.runs[0];

export const runningRun: CanvasesCanvasRun = synthesizeRun(
  newestPassed,
  { state: "STATE_STARTED", result: "RESULT_UNKNOWN", finishedAt: undefined },
  "running",
  45_000,
);

export const cancelledRun: CanvasesCanvasRun = synthesizeRun(
  newestPassed,
  { state: "STATE_FINISHED", result: "RESULT_CANCELLED" },
  "cancelled",
  3 * 60 * 60_000,
);

export const allOutcomeRuns: CanvasesCanvasRun[] = [
  runningRun,
  ...data.runs.filter((run) => run.result === "RESULT_PASSED").slice(0, 5),
  ...data.runs.filter((run) => run.result === "RESULT_FAILED").slice(0, 5),
  cancelledRun,
];
