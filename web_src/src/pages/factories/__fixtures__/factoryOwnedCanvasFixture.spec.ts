import { describe, expect, it } from "vitest";

import { canvasAppIds } from "@/pages/app/__fixtures__/handlers";
import { isValidRunId } from "@/pages/app/workflowPageHelpers";

import {
  factoryOwnedCanvasFixture,
  REFUND_IMPLEMENTER_APP,
  refundFactoryLineRuns,
  refundLineCanvasFixture,
} from "./factoryOwnedCanvasFixture";
import {
  LINE_RUN_IMPLEMENT_FAILED_ID,
  LINE_RUN_IMPLEMENT_FAILED_ROOT_EVENT_ID,
  LINE_RUN_IMPLEMENT_ID,
  LINE_RUN_IMPLEMENT_PASSED_ID,
  LINE_RUN_VERIFY_PASSED_ID,
  PRIMARY_FACTORY_ID,
} from "./factoryPageResponses";
import { SIMPLE_FACTORY_RUN_EDGES, SIMPLE_FACTORY_RUN_NODE_IDS } from "./simpleFactoryRunCanvas";

describe("factoryOwnedCanvasFixture", () => {
  it("marks the canvas as owned by the Refunds Factory", () => {
    const fixture = factoryOwnedCanvasFixture(REFUND_IMPLEMENTER_APP);

    expect(fixture.canvasId).toBe(REFUND_IMPLEMENTER_APP.id);
    expect(fixture.canvas?.canvas?.metadata).toMatchObject({
      id: REFUND_IMPLEMENTER_APP.id,
      name: REFUND_IMPLEMENTER_APP.name,
      factoryId: PRIMARY_FACTORY_ID,
    });
  });

  it("includes UUID canvas runs that Line cards open", () => {
    const fixture = refundLineCanvasFixture();
    const runIds = (refundFactoryLineRuns().runs ?? []).map((run) => run.id);

    expect(fixture.runs?.runs?.map((run) => run.id)).toEqual(runIds);
    expect(runIds).toEqual([
      LINE_RUN_IMPLEMENT_ID,
      LINE_RUN_IMPLEMENT_FAILED_ID,
      LINE_RUN_IMPLEMENT_PASSED_ID,
      LINE_RUN_VERIFY_PASSED_ID,
    ]);
    expect(runIds.every((runId) => typeof runId === "string" && isValidRunId(runId))).toBe(true);
  });

  it("reuses the captured published run for the failed implement card", () => {
    const failedImplement = (refundFactoryLineRuns().runs ?? []).find((run) => run.id === LINE_RUN_IMPLEMENT_FAILED_ID);

    expect(LINE_RUN_IMPLEMENT_FAILED_ID).toBe(canvasAppIds.publishedRunId);
    expect(failedImplement?.rootEvent).toMatchObject({
      id: LINE_RUN_IMPLEMENT_FAILED_ROOT_EVENT_ID,
      nodeId: "on-run",
    });
    expect(LINE_RUN_IMPLEMENT_FAILED_ROOT_EVENT_ID).toBe(canvasAppIds.rootEventId);
  });

  it("inspects a compact CI graph instead of the Software Factory capture", () => {
    const fixture = refundLineCanvasFixture();
    const spec = fixture.canvas?.canvas?.spec as {
      nodes?: Array<{ id?: string }>;
      edges?: Array<{ sourceId?: string; targetId?: string; channel?: string }>;
    };
    const executions = fixture.executionsByEventId?.[LINE_RUN_IMPLEMENT_FAILED_ROOT_EVENT_ID]?.executions ?? [];
    const actionNames = ((fixture.actions as { actions?: Array<{ name?: string }> } | undefined)?.actions ?? []).map(
      (action) => action.name,
    );

    expect(fixture.canvas?.canvas?.metadata).toMatchObject({ factoryId: PRIMARY_FACTORY_ID });
    expect(spec.nodes?.map((node) => node.id)).toEqual([...SIMPLE_FACTORY_RUN_NODE_IDS]);
    expect(spec.edges).toEqual(SIMPLE_FACTORY_RUN_EDGES.map((edge) => ({ ...edge })));
    expect(executions.map((execution) => (execution as { nodeId?: string }).nodeId)).toEqual([
      "loop",
      "mark-pr-ready",
      "run-workflow",
    ]);
    expect(actionNames).toEqual(
      expect.arrayContaining([
        "loop",
        "github.markPullRequestReadyForReview",
        "semaphore.runWorkflow",
        "runnerClaudeCode",
      ]),
    );
  });
});
