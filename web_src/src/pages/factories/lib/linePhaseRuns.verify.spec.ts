import { describe, expect, it } from "bun:test";
import type { FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";
import { buildLinePhaseBoard, collectLineVerifyOrders, visibleLineStageColumns } from "./linePhaseRuns";

const LINE: FactoriesFactoryLine = {
  id: "line-1",
  steps: [{ app: { app: "app-build" } }],
  stages: [
    { id: "verify", key: "verify", name: "Review", kind: "STAGE_KIND_VERIFY" },
    { id: "phase-0", key: "phase-0", name: "Build", kind: "STAGE_KIND_PHASE", stepIndex: 0 },
  ],
};

function order(id: string, reviewReady: boolean): FactoriesWorkOrder {
  return {
    id,
    state: "STATE_OPEN",
    lineDispatches: [
      {
        id: `dispatch-${id}`,
        line: { id: LINE.id },
        reviewReady,
        stepExecutions: [{ id: `execution-${id}`, stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_PASSED" }],
      },
    ],
  };
}

describe("collectLineVerifyOrders", () => {
  it("uses stored readiness with a renamed and reordered Verify stage", () => {
    const ready = order("ready", true);
    const notReady = order("not-ready", false);
    const board = buildLinePhaseBoard(LINE, [ready, notReady]);
    const verify = collectLineVerifyOrders([ready, notReady], LINE);
    expect(verify.map((entry) => entry.id)).toEqual(["ready"]);
    expect(
      visibleLineStageColumns(board, verify).flatMap((column) => column.runs.map((run) => run.workOrderId)),
    ).toEqual(["not-ready"]);
    expect(visibleLineStageColumns(board, verify, false).flatMap((column) => column.runs)).toHaveLength(2);
  });

  it("does not require an execution for a direct handoff", () => {
    const ready = order("handoff", true);
    ready.lineDispatches![0].stepExecutions = [];
    expect(collectLineVerifyOrders([ready], LINE)).toEqual([ready]);
  });

  it("uses the latest traversal on this line, not an older ready traversal", () => {
    const ready = order("rerun", true);
    ready.lineDispatches![0].createdAt = "2026-08-11T12:00:00Z";
    ready.lineDispatches!.push({
      id: "new",
      line: { id: LINE.id },
      createdAt: "2026-08-11T13:00:00Z",
      reviewReady: false,
    });
    expect(collectLineVerifyOrders([ready], LINE)).toEqual([]);
    ready.lineDispatches![1].line = { id: "other-line" };
    expect(collectLineVerifyOrders([ready], LINE)).toEqual([ready]);
  });

  it("gives lifecycle states precedence and preserves source ordering", () => {
    const first = order("first", true);
    const second = { ...order("second", true), updatedAt: "2026-08-11T14:00:00Z" };
    const draft = { ...order("draft", true), state: "STATE_DRAFT" as const };
    const closed = { ...order("closed", true), state: "STATE_CLOSED" as const };
    const archived = { ...order("archived", true), result: "RESULT_ARCHIVED" as const };
    expect(collectLineVerifyOrders([first, second, draft, closed, archived], LINE)).toEqual([first, second]);
  });
});
