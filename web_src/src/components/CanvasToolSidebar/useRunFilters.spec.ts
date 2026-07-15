import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import type { CanvasesCanvasRun, SuperplaneComponentsNode } from "@/api-client";

import { RUNS_SIDEBAR_FILTERS_STORAGE_KEY } from "./filterPersistence";
import { useRunFilters } from "./useRunFilters";

const NODES: SuperplaneComponentsNode[] = [
  { id: "trigger-a", name: "deploy", type: "TYPE_TRIGGER", component: "github" },
  { id: "trigger-b", name: "release", type: "TYPE_TRIGGER", component: "github" },
];

function run(overrides: Partial<CanvasesCanvasRun>): CanvasesCanvasRun {
  return {
    id: "run",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    rootEvent: { nodeId: "trigger-a" },
    ...overrides,
  };
}

const RUNS: CanvasesCanvasRun[] = [
  run({ id: "passed-a", result: "RESULT_PASSED", rootEvent: { nodeId: "trigger-a" } }),
  run({ id: "failed-a", result: "RESULT_FAILED", rootEvent: { nodeId: "trigger-a" } }),
  run({ id: "failed-b", result: "RESULT_FAILED", rootEvent: { nodeId: "trigger-b" } }),
  run({ id: "running-b", state: "STATE_STARTED", result: undefined, rootEvent: { nodeId: "trigger-b" } }),
];

describe("useRunFilters", () => {
  beforeEach(() => {
    window.localStorage.removeItem(RUNS_SIDEBAR_FILTERS_STORAGE_KEY);
  });

  it("filters by status through the shared runMatchesStatusTriggerFilters matcher", () => {
    const { result } = renderHook(() => useRunFilters({ runs: RUNS, workflowNodes: NODES, componentIconMap: {} }));

    act(() => {
      result.current.toggleStatus("failed");
    });

    expect(result.current.filteredRuns.map((item) => item.run.id)).toEqual(["failed-a", "failed-b"]);
  });

  it("filters by trigger through the shared matcher", () => {
    const { result } = renderHook(() => useRunFilters({ runs: RUNS, workflowNodes: NODES, componentIconMap: {} }));

    act(() => {
      result.current.toggleTrigger("trigger-b");
    });

    expect(result.current.filteredRuns.map((item) => item.run.id)).toEqual(["failed-b", "running-b"]);
  });

  it("ANDs status and trigger filters the same way as console run datasources", () => {
    const { result } = renderHook(() => useRunFilters({ runs: RUNS, workflowNodes: NODES, componentIconMap: {} }));

    act(() => {
      result.current.toggleStatus("failed");
      result.current.toggleTrigger("trigger-a");
    });

    expect(result.current.filteredRuns.map((item) => item.run.id)).toEqual(["failed-a"]);
  });

  it("drops unknown-status runs when a status filter is active", () => {
    const runs = [run({ id: "unknown", state: undefined, result: undefined })];
    const { result } = renderHook(() => useRunFilters({ runs, workflowNodes: NODES, componentIconMap: {} }));

    act(() => {
      result.current.toggleStatus("passed");
    });

    expect(result.current.filteredRuns).toHaveLength(0);
  });
});
