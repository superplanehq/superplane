import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { isRunDetailDismissed, runInspectorAutoOpenStorageKey, useRunsDetailState } from "./useRunsDetailState";

function makeSearchParams(params: Record<string, string> = {}) {
  return new URLSearchParams(params);
}

describe("useRunsDetailState", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("opens run detail on mount when run is in the URL", () => {
    const { result } = renderHook(() => useRunsDetailState(makeSearchParams({ run: "run-1" }), true, "run-1"));

    expect(result.current.openRunDetailOnMount).toBe(true);
  });

  it("respects persisted closed detail on mount when run is in the URL", () => {
    window.localStorage.setItem(runInspectorAutoOpenStorageKey("canvas-1"), "false");

    const { result } = renderHook(() =>
      useRunsDetailState(makeSearchParams({ run: "run-1" }), true, "run-1", undefined, { canvasId: "canvas-1" }),
    );

    expect(result.current.openRunDetailOnMount).toBe(false);
    expect(isRunDetailDismissed(result.current.detailDismissedForRunId, "run-1")).toBe(true);
  });

  it("restores the selected run node on mount when the URL points at the detail pane", () => {
    const { result } = renderHook(() =>
      useRunsDetailState(makeSearchParams({ run: "run-1", sidebar: "1", node: "node-1" }), true, "run-1"),
    );

    expect(result.current.runDetailNodeId).toBe("node-1");
  });

  it("does not restore the selected run node without the detail pane flag", () => {
    const { result } = renderHook(() =>
      useRunsDetailState(makeSearchParams({ run: "run-1", node: "node-1" }), true, "run-1"),
    );

    expect(result.current.runDetailNodeId).toBeNull();
  });

  it("restores the selected run node when the URL changes to another run detail", () => {
    const { result, rerender } = renderHook(
      ({ searchParams, selectedRunId }) => useRunsDetailState(searchParams, true, selectedRunId),
      {
        initialProps: {
          searchParams: makeSearchParams({ run: "run-1" }),
          selectedRunId: "run-1" as string | null,
        },
      },
    );

    expect(result.current.runDetailNodeId).toBeNull();

    rerender({
      searchParams: makeSearchParams({ run: "run-2", sidebar: "1", node: "node-2" }),
      selectedRunId: "run-2",
    });

    expect(result.current.runDetailNodeId).toBe("node-2");
  });

  it("updates the selected run node when browser navigation changes the node URL param", () => {
    const { result, rerender } = renderHook(
      ({ searchParams, selectedRunId }) => useRunsDetailState(searchParams, true, selectedRunId),
      {
        initialProps: {
          searchParams: makeSearchParams({ run: "run-1", sidebar: "1", node: "node-1" }),
          selectedRunId: "run-1" as string | null,
        },
      },
    );

    expect(result.current.runDetailNodeId).toBe("node-1");

    rerender({
      searchParams: makeSearchParams({ run: "run-1", sidebar: "1", node: "node-2" }),
      selectedRunId: "run-1",
    });

    expect(result.current.runDetailNodeId).toBe("node-2");
  });

  it("clears the selected run node when browser navigation removes the node URL param", () => {
    const { result, rerender } = renderHook(
      ({ searchParams, selectedRunId }) => useRunsDetailState(searchParams, true, selectedRunId),
      {
        initialProps: {
          searchParams: makeSearchParams({ run: "run-1", sidebar: "1", node: "node-1" }),
          selectedRunId: "run-1" as string | null,
        },
      },
    );

    expect(result.current.runDetailNodeId).toBe("node-1");

    rerender({
      searchParams: makeSearchParams({ run: "run-1", sidebar: "1" }),
      selectedRunId: "run-1",
    });

    expect(result.current.runDetailNodeId).toBeNull();
  });

  it("clears openRunDetailOnMount when leaving run inspection", () => {
    const { result, rerender } = renderHook(
      ({ isRunInspectionMode, searchParams, selectedRunId }) =>
        useRunsDetailState(searchParams, isRunInspectionMode, selectedRunId),
      {
        initialProps: {
          isRunInspectionMode: true,
          searchParams: makeSearchParams({ run: "run-1" }),
          selectedRunId: "run-1" as string | null,
        },
      },
    );

    expect(result.current.openRunDetailOnMount).toBe(true);

    rerender({
      isRunInspectionMode: false,
      searchParams: makeSearchParams({ run: "run-1" }),
      selectedRunId: "run-1",
    });

    expect(result.current.openRunDetailOnMount).toBe(false);
  });

  it("does not reopen detail after leaving and re-entering run inspection without a run param", () => {
    const { result, rerender } = renderHook(
      ({ isRunInspectionMode, searchParams, selectedRunId }) =>
        useRunsDetailState(searchParams, isRunInspectionMode, selectedRunId),
      {
        initialProps: {
          isRunInspectionMode: true,
          searchParams: makeSearchParams({ run: "run-1" }),
          selectedRunId: "run-1" as string | null,
        },
      },
    );

    rerender({
      isRunInspectionMode: false,
      searchParams: makeSearchParams({ run: "run-1" }),
      selectedRunId: "run-1",
    });

    rerender({
      isRunInspectionMode: true,
      searchParams: makeSearchParams(),
      selectedRunId: null,
    });

    expect(result.current.openRunDetailOnMount).toBe(false);
  });

  it("reopens detail when entering run inspection with a run param that was not dismissed", () => {
    const { result, rerender } = renderHook(
      ({ isRunInspectionMode, searchParams, selectedRunId }) =>
        useRunsDetailState(searchParams, isRunInspectionMode, selectedRunId),
      {
        initialProps: {
          isRunInspectionMode: false,
          searchParams: makeSearchParams({ run: "run-1" }),
          selectedRunId: "run-1" as string | null,
        },
      },
    );

    rerender({
      isRunInspectionMode: true,
      searchParams: makeSearchParams({ run: "run-2" }),
      selectedRunId: "run-2",
    });

    expect(result.current.openRunDetailOnMount).toBe(true);
  });

  it("clears the selected run node when the run changes", () => {
    const { result, rerender } = renderHook(
      ({ isRunInspectionMode, searchParams, selectedRunId }) =>
        useRunsDetailState(searchParams, isRunInspectionMode, selectedRunId),
      {
        initialProps: {
          isRunInspectionMode: true,
          searchParams: makeSearchParams({ run: "run-1" }),
          selectedRunId: "run-1" as string | null,
        },
      },
    );

    act(() => {
      result.current.setRunDetailNodeId("node-1");
    });
    expect(result.current.runDetailNodeId).toBe("node-1");

    rerender({
      isRunInspectionMode: true,
      searchParams: makeSearchParams({ run: "run-2" }),
      selectedRunId: "run-2",
    });

    expect(result.current.runDetailNodeId).toBeNull();
  });

  it("preserves the selected run node when flagged before changing runs", () => {
    const preserveRunDetailNodeOnNextRunChangeRef = { current: false };
    const { result, rerender } = renderHook(
      ({ isRunInspectionMode, searchParams, selectedRunId }) =>
        useRunsDetailState(searchParams, isRunInspectionMode, selectedRunId, preserveRunDetailNodeOnNextRunChangeRef),
      {
        initialProps: {
          isRunInspectionMode: true,
          searchParams: makeSearchParams({ run: "run-1" }),
          selectedRunId: "run-1" as string | null,
        },
      },
    );

    act(() => {
      result.current.setRunDetailNodeId("node-1");
    });

    preserveRunDetailNodeOnNextRunChangeRef.current = true;
    rerender({
      isRunInspectionMode: true,
      searchParams: makeSearchParams({ run: "run-2" }),
      selectedRunId: "run-2",
    });

    expect(result.current.runDetailNodeId).toBe("node-1");
  });

  it("keeps detail closed when the user dismissed detail for the run in the URL", () => {
    const onBackToRunList = vi.fn();
    const { result, rerender } = renderHook(
      ({ isRunInspectionMode, searchParams, selectedRunId }) =>
        useRunsDetailState(searchParams, isRunInspectionMode, selectedRunId, undefined, { onBackToRunList }),
      {
        initialProps: {
          isRunInspectionMode: true,
          searchParams: makeSearchParams({ run: "run-1" }),
          selectedRunId: "run-1" as string | null,
        },
      },
    );

    act(() => {
      result.current.handleBackToRunList();
    });

    expect(result.current.openRunDetailOnMount).toBe(false);

    rerender({
      isRunInspectionMode: false,
      searchParams: makeSearchParams({ run: "run-1" }),
      selectedRunId: "run-1",
    });

    rerender({
      isRunInspectionMode: true,
      searchParams: makeSearchParams({ run: "run-1" }),
      selectedRunId: "run-1",
    });

    expect(result.current.openRunDetailOnMount).toBe(false);
    expect(onBackToRunList).toHaveBeenCalledOnce();
  });

  it("lets URL state reopen run detail for a run that was dismissed locally", async () => {
    const onBackToRunList = vi.fn();
    const { result, rerender } = renderHook(
      ({ searchParams }) =>
        useRunsDetailState(searchParams, true, "run-1", undefined, {
          onBackToRunList,
        }),
      {
        initialProps: {
          searchParams: new URLSearchParams("run=run-1"),
        },
      },
    );

    act(() => {
      result.current.handleBackToRunList();
    });

    expect(result.current.detailDismissedForRunId).toBe("run-1");
    expect(result.current.runDetailNodeId).toBeNull();
    expect(onBackToRunList).toHaveBeenCalledOnce();

    rerender({
      searchParams: new URLSearchParams("run=run-1&sidebar=1&node=node-1"),
    });

    await waitFor(() => {
      expect(result.current.detailDismissedForRunId).toBeNull();
    });
    expect(result.current.runDetailNodeId).toBe("node-1");
  });

  it("opens run detail for run row selection by default", () => {
    const { result } = renderHook(() =>
      useRunsDetailState(makeSearchParams({ run: "run-1" }), true, "run-1", undefined, { canvasId: "canvas-1" }),
    );

    act(() => {
      result.current.maybeOpenRunDetailForRun("run-1");
    });

    expect(result.current.detailDismissedForRunId).toBeNull();
  });

  it("reopens run detail when a run row is selected after dismissal", () => {
    const { result } = renderHook(() =>
      useRunsDetailState(makeSearchParams({ run: "run-1" }), true, "run-1", undefined, { canvasId: "canvas-1" }),
    );

    act(() => {
      result.current.handleBackToRunList();
    });

    expect(window.localStorage.getItem(runInspectorAutoOpenStorageKey("canvas-1"))).toBe("false");

    act(() => {
      result.current.clearDismissedRunDetail({ persistAutoOpen: true });
    });

    expect(window.localStorage.getItem(runInspectorAutoOpenStorageKey("canvas-1"))).toBe("true");
    expect(isRunDetailDismissed(result.current.detailDismissedForRunId, "run-1")).toBe(false);
    expect(isRunDetailDismissed(result.current.detailDismissedForRunId, "run-2")).toBe(false);
  });

  it("persists auto-open again when a node opens run detail", () => {
    const { result } = renderHook(() =>
      useRunsDetailState(makeSearchParams({ run: "run-1" }), true, "run-1", undefined, { canvasId: "canvas-1" }),
    );

    act(() => {
      result.current.handleBackToRunList();
    });
    act(() => {
      result.current.clearDismissedRunDetail({ persistAutoOpen: true });
    });
    act(() => {
      result.current.maybeOpenRunDetailForRun("run-2");
    });

    expect(window.localStorage.getItem(runInspectorAutoOpenStorageKey("canvas-1"))).toBe("true");
    expect(result.current.detailDismissedForRunId).toBeNull();
  });
});
