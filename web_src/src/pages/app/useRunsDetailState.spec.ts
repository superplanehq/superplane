import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useRunsDetailState } from "./useRunsDetailState";

function makeSearchParams(params: Record<string, string> = {}) {
  return new URLSearchParams(params);
}

describe("useRunsDetailState", () => {
  it("opens run detail on mount when run is in the URL", () => {
    const { result } = renderHook(() => useRunsDetailState(makeSearchParams({ run: "run-1" }), true, "run-1"));

    expect(result.current.openRunDetailOnMount).toBe(true);
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
        useRunsDetailState(searchParams, isRunInspectionMode, selectedRunId, undefined, onBackToRunList),
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
});
