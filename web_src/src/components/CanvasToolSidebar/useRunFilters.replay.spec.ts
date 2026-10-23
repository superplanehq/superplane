import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import type { CanvasesCanvasRun } from "@/api-client";
import { useRunFilters } from "./useRunFilters";

function makeRun(id: string, overrides: Partial<CanvasesCanvasRun> = {}): CanvasesCanvasRun {
  return {
    id,
    canvasId: "canvas-1",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    ...overrides,
  };
}

function renderRunFilters(runs: CanvasesCanvasRun[], selectedRunId: string | null = null) {
  return renderHook(() =>
    useRunFilters({
      runs,
      workflowNodes: [],
      componentIconMap: {},
      selectedRunId,
    }),
  );
}

function visibleRunIds(result: { current: ReturnType<typeof useRunFilters> }): (string | undefined)[] {
  return result.current.filteredRuns.map(({ run }) => run.id);
}

describe("useRunFilters — replay runs", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("hides replay runs by default", () => {
    const { result } = renderRunFilters([makeRun("run-normal"), makeRun("run-replay", { isReplay: true })]);

    expect(visibleRunIds(result)).toEqual(["run-normal"]);
    expect(result.current.showReplays).toBe(false);
  });

  it("shows replay runs once the filter is enabled, and hides them again when disabled", () => {
    const { result } = renderRunFilters([makeRun("run-normal"), makeRun("run-replay", { isReplay: true })]);

    act(() => result.current.toggleShowReplays());

    expect(result.current.showReplays).toBe(true);
    expect(visibleRunIds(result)).toEqual(["run-normal", "run-replay"]);

    act(() => result.current.toggleShowReplays());

    expect(visibleRunIds(result)).toEqual(["run-normal"]);
  });

  it("leaves a list containing no replay runs unchanged in both states (boundary)", () => {
    const { result } = renderRunFilters([makeRun("run-a"), makeRun("run-b")]);

    expect(visibleRunIds(result)).toEqual(["run-a", "run-b"]);

    act(() => result.current.toggleShowReplays());

    expect(visibleRunIds(result)).toEqual(["run-a", "run-b"]);
  });

  it("keeps the selected replay run listed even while replays are hidden", () => {
    const { result } = renderRunFilters(
      [makeRun("run-normal"), makeRun("run-replay", { isReplay: true })],
      "run-replay",
    );

    expect(result.current.showReplays).toBe(false);
    expect(visibleRunIds(result)).toEqual(["run-normal", "run-replay"]);
  });

  it("still hides an unselected replay run when a different run is selected (boundary)", () => {
    const { result } = renderRunFilters(
      [makeRun("run-normal"), makeRun("run-replay", { isReplay: true })],
      "run-normal",
    );

    expect(visibleRunIds(result)).toEqual(["run-normal"]);
  });

  it("counts the replay runs it is hiding", () => {
    const { result } = renderRunFilters([
      makeRun("run-normal"),
      makeRun("run-replay-1", { isReplay: true }),
      makeRun("run-replay-2", { isReplay: true }),
    ]);

    expect(result.current.hiddenReplayCount).toBe(2);
  });

  it("counts nothing once replays are shown (boundary)", () => {
    const { result } = renderRunFilters([makeRun("run-normal"), makeRun("run-replay", { isReplay: true })]);

    act(() => result.current.toggleShowReplays());

    expect(result.current.hiddenReplayCount).toBe(0);
  });

  it("does not count the selected replay run, which stays listed (boundary)", () => {
    const { result } = renderRunFilters(
      [makeRun("run-normal"), makeRun("run-replay", { isReplay: true })],
      "run-replay",
    );

    expect(result.current.hiddenReplayCount).toBe(0);
  });

  it("does not count a replay run another filter already excludes (boundary)", () => {
    const { result } = renderRunFilters([
      makeRun("run-normal"),
      makeRun("run-replay", { isReplay: true, state: "STATE_STARTED", result: "RESULT_UNKNOWN" }),
    ]);

    act(() => result.current.toggleStatus("passed"));

    expect(result.current.hiddenReplayCount).toBe(0);
  });

  it("returns replay runs to hidden when filters are cleared", () => {
    const { result } = renderRunFilters([makeRun("run-normal"), makeRun("run-replay", { isReplay: true })]);

    act(() => result.current.toggleShowReplays());
    expect(visibleRunIds(result)).toEqual(["run-normal", "run-replay"]);

    act(() => result.current.clearFilters());

    expect(result.current.showReplays).toBe(false);
    expect(visibleRunIds(result)).toEqual(["run-normal"]);
  });
});
