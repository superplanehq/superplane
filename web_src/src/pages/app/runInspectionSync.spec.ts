import { describe, expect, it } from "vitest";
import {
  clearRunDetailNodeSearchParams,
  hasLoadedAllRuns,
  shouldClearRunDetailNode,
  shouldClearStaleRunUrl,
} from "./runInspectionSync";

describe("runInspectionSync", () => {
  it("detects when all runs are loaded", () => {
    expect(hasLoadedAllRuns([{ runs: [{ id: "run-1" }], totalCount: 1 }], false)).toBe(true);
    expect(hasLoadedAllRuns([{ runs: [{ id: "run-1" }], totalCount: 2 }], true)).toBe(false);
    expect(hasLoadedAllRuns([{ runs: [{ id: "run-1" }], totalCount: 2 }], false)).toBe(true);
    expect(hasLoadedAllRuns([{ runs: [{ id: "run-1" }] }], true)).toBe(false);
    expect(hasLoadedAllRuns([{ runs: [{ id: "run-1" }] }], false)).toBe(true);
  });

  it("clears stale run URLs only after the run list finishes loading", () => {
    expect(
      shouldClearStaleRunUrl({
        selectedRunId: "missing-run",
        isRunInspectionMode: true,
        selectedRun: null,
        isRunsQueryLoading: true,
        isFetchingNextPage: false,
        pages: [],
        hasNextPage: true,
      }),
    ).toBe(false);

    expect(
      shouldClearStaleRunUrl({
        selectedRunId: "missing-run",
        isRunInspectionMode: true,
        selectedRun: null,
        isRunsQueryLoading: false,
        isFetchingNextPage: false,
        pages: [{ runs: [], totalCount: 0 }],
        hasNextPage: false,
      }),
    ).toBe(true);
  });

  it("clears run detail nodes that are not part of the selected run", () => {
    expect(
      shouldClearRunDetailNode({
        runDetailNodeId: "node-b",
        participantNodeIds: ["node-a"],
        runCanvasLoading: false,
        runCanvasReady: true,
      }),
    ).toBe(true);

    expect(
      shouldClearRunDetailNode({
        runDetailNodeId: "node-a",
        participantNodeIds: ["node-a"],
        runCanvasLoading: false,
        runCanvasReady: true,
      }),
    ).toBe(false);

    expect(
      shouldClearRunDetailNode({
        runDetailNodeId: "node-a",
        participantNodeIds: [],
        runCanvasLoading: false,
        runCanvasReady: true,
      }),
    ).toBe(true);

    expect(
      shouldClearRunDetailNode({
        runDetailNodeId: "node-a",
        participantNodeIds: [],
        runCanvasLoading: true,
        runCanvasReady: false,
      }),
    ).toBe(false);

    expect(
      shouldClearRunDetailNode({
        runDetailNodeId: "node-a",
        participantNodeIds: [],
        runCanvasLoading: false,
        runCanvasReady: false,
      }),
    ).toBe(false);
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
