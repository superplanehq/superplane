import { describe, expect, it } from "vitest";
import {
  clearRunDetailNodeSearchParams,
  isUnresolvableRunError,
  isValidRunId,
  shouldClearRunDetailNode,
  shouldClearStaleRunUrl,
} from "./workflowPageHelpers";

describe("workflowPageHelpers run inspection", () => {
  it("clears stale run URLs only after describe run resolves as unresolvable", () => {
    expect(
      shouldClearStaleRunUrl({
        selectedRunId: "missing-run",
        isRunInspectionMode: true,
        selectedRun: null,
        isRunResolveLoading: true,
        isRunUnresolvable: false,
      }),
    ).toBe(false);

    expect(
      shouldClearStaleRunUrl({
        selectedRunId: "missing-run",
        isRunInspectionMode: true,
        selectedRun: null,
        isRunResolveLoading: false,
        isRunUnresolvable: true,
      }),
    ).toBe(true);
  });

  it("treats malformed run ids as unresolvable", () => {
    expect(isValidRunId("not-a-uuid")).toBe(false);
    expect(
      shouldClearStaleRunUrl({
        selectedRunId: "not-a-uuid",
        isRunInspectionMode: true,
        selectedRun: null,
        isRunResolveLoading: false,
        isRunUnresolvable: true,
      }),
    ).toBe(true);
  });

  it("treats invalid argument describe errors as unresolvable", () => {
    expect(isUnresolvableRunError({ code: "INVALID_ARGUMENT" })).toBe(true);
    expect(isUnresolvableRunError({ status: 400 })).toBe(true);
    expect(isUnresolvableRunError({ status: 500 })).toBe(false);
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
