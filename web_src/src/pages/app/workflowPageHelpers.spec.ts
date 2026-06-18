import { describe, expect, it } from "vitest";
import {
  clearRunDetailNodeSearchParams,
  isValidRunId,
  shouldClearRunDetailNode,
  shouldClearStaleRunUrl,
} from "./workflowPageHelpers";

const validRunId = "550e8400-e29b-41d4-a716-446655440000";

describe("workflowPageHelpers run inspection", () => {
  it("clears stale run URLs after describe settles without a run", () => {
    expect(
      shouldClearStaleRunUrl({
        selectedRunId: validRunId,
        isRunInspectionMode: true,
        selectedRun: null,
        isRunResolveLoading: true,
        describeRunSettled: false,
      }),
    ).toBe(false);

    expect(
      shouldClearStaleRunUrl({
        selectedRunId: validRunId,
        isRunInspectionMode: true,
        selectedRun: null,
        isRunResolveLoading: false,
        describeRunSettled: true,
      }),
    ).toBe(true);
  });

  it("clears malformed run ids immediately", () => {
    expect(isValidRunId("not-a-uuid")).toBe(false);
    expect(
      shouldClearStaleRunUrl({
        selectedRunId: "not-a-uuid",
        isRunInspectionMode: true,
        selectedRun: null,
        isRunResolveLoading: false,
        describeRunSettled: false,
      }),
    ).toBe(true);
  });

  it("does not clear when the run resolved", () => {
    expect(
      shouldClearStaleRunUrl({
        selectedRunId: validRunId,
        isRunInspectionMode: true,
        selectedRun: { id: validRunId },
        isRunResolveLoading: false,
        describeRunSettled: true,
      }),
    ).toBe(false);
  });

  it("clears run detail nodes that are not part of the selected run", () => {
    expect(
      shouldClearRunDetailNode({
        runDetailNodeId: "node-b",
        participantNodeIds: ["node-a"],
        runCanvasLoading: false,
        runCanvasSettled: true,
      }),
    ).toBe(true);

    expect(
      shouldClearRunDetailNode({
        runDetailNodeId: "node-a",
        participantNodeIds: ["node-a"],
        runCanvasLoading: false,
        runCanvasSettled: true,
      }),
    ).toBe(false);

    expect(
      shouldClearRunDetailNode({
        runDetailNodeId: "node-a",
        participantNodeIds: [],
        runCanvasLoading: false,
        runCanvasSettled: true,
      }),
    ).toBe(true);

    expect(
      shouldClearRunDetailNode({
        runDetailNodeId: "node-a",
        participantNodeIds: [],
        runCanvasLoading: true,
        runCanvasSettled: false,
      }),
    ).toBe(false);

    expect(
      shouldClearRunDetailNode({
        runDetailNodeId: "node-a",
        participantNodeIds: [],
        runCanvasLoading: false,
        runCanvasSettled: false,
      }),
    ).toBe(false);

    expect(
      shouldClearRunDetailNode({
        runDetailNodeId: "node-a",
        participantNodeIds: [],
        runCanvasLoading: false,
        runCanvasSettled: true,
      }),
    ).toBe(true);
  });

  it("clears matching stale run detail node search params", () => {
    const cleared = clearRunDetailNodeSearchParams(
      new URLSearchParams({ run: "run-1", sidebar: "1", node: "node-a" }),
      "node-a",
    );

    expect(cleared.get("run")).toBe("run-1");
    expect(cleared.get("sidebar")).toBeNull();
    expect(cleared.get("node")).toBeNull();
  });

  it("keeps run detail node search params for a newer URL selection", () => {
    const unchanged = clearRunDetailNodeSearchParams(
      new URLSearchParams({ run: "run-1", sidebar: "1", node: "node-b" }),
      "node-a",
    );

    expect(unchanged.get("sidebar")).toBe("1");
    expect(unchanged.get("node")).toBe("node-b");
  });
});
