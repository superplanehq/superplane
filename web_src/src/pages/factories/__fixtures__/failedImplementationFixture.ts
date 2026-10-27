import { factoryOwnedCanvasFixture, REFUND_IMPLEMENTER_APP } from "./factoryOwnedCanvasFixture";
import { HOUR_AGO, TWO_HOURS_AGO, LINE_RUN_IMPLEMENT_FAILED_ID } from "./factoryPageResponses";

/** The agent passed, but attaching its pull request failed. */
export function failedImplementationCanvasFixture() {
  const canvasId = REFUND_IMPLEMENTER_APP.id;
  const runId = LINE_RUN_IMPLEMENT_FAILED_ID;
  const agent = {
    id: "failed-story-agent",
    runId,
    nodeId: "implement",
    componentName: "runnerSuperPlane",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: TWO_HOURS_AGO,
    updatedAt: HOUR_AGO,
    outputs: { passed: [{ data: { result: "Changes committed and pushed." } }] },
  };
  const attach = {
    id: "failed-story-attach",
    runId,
    nodeId: "attach-pr",
    componentName: "addWorkOrderPullRequest",
    state: "STATE_FINISHED",
    result: "RESULT_FAILED",
    createdAt: HOUR_AGO,
    updatedAt: HOUR_AGO,
    resultMessage: "factory pull request already exists",
  };
  const run = {
    id: runId,
    canvasId,
    versionId: "failure-version",
    state: "STATE_FINISHED",
    result: "RESULT_FAILED",
    createdAt: TWO_HOURS_AGO,
    finishedAt: HOUR_AGO,
    task: { id: "wo-board-implement-failed", key: "RF-106", title: "Fix refund dispatcher timeout loop" },
    rootEvent: { id: "failure-event", data: { workOrder: { id: "wo-board-implement-failed", number: 106 } } },
    executions: [agent, attach],
  };
  return factoryOwnedCanvasFixture(REFUND_IMPLEMENTER_APP, {
    runs: { runs: [run], totalCount: 1 },
    runDetailsById: { [runId]: { run } },
    executionsByEventId: { "failure-event": { executions: [agent, attach] } },
    canvas: {
      canvas: {
        spec: {
          nodes: [
            { id: "on-run", name: "Source event", type: "TYPE_TRIGGER", component: "onRun" },
            { id: "implement", name: "Implement task", type: "TYPE_ACTION", component: "runnerSuperPlane" },
            { id: "attach-pr", name: "Attach PR to task", type: "TYPE_ACTION", component: "addWorkOrderPullRequest" },
          ],
          edges: [
            { sourceId: "on-run", targetId: "implement", channel: "default" },
            { sourceId: "implement", targetId: "attach-pr", channel: "passed" },
          ],
        },
      },
    },
  });
}
