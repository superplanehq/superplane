import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useRunsDetailState } from "./useRunsDetailState";

function makeSearchParams(params: Record<string, string> = {}) {
  return new URLSearchParams(params);
}

describe("useRunsDetailState", () => {
  it("opens run detail on mount when run is in the URL", () => {
    const { result } = renderHook(() => useRunsDetailState(makeSearchParams({ run: "run-1" }), true, "run-1"));

    expect(result.current.openRunDetailOnMount).toBe(true);
  });

  it("clears openRunDetailOnMount when leaving runs mode", () => {
    const { result, rerender } = renderHook(
      ({ isRunsMode, searchParams, selectedRunId }) => useRunsDetailState(searchParams, isRunsMode, selectedRunId),
      {
        initialProps: {
          isRunsMode: true,
          searchParams: makeSearchParams({ run: "run-1" }),
          selectedRunId: "run-1" as string | null,
        },
      },
    );

    expect(result.current.openRunDetailOnMount).toBe(true);

    rerender({
      isRunsMode: false,
      searchParams: makeSearchParams({ run: "run-1" }),
      selectedRunId: "run-1",
    });

    expect(result.current.openRunDetailOnMount).toBe(false);
  });

  it("does not reopen detail after leaving and re-entering runs without a run param", () => {
    const { result, rerender } = renderHook(
      ({ isRunsMode, searchParams, selectedRunId }) => useRunsDetailState(searchParams, isRunsMode, selectedRunId),
      {
        initialProps: {
          isRunsMode: true,
          searchParams: makeSearchParams({ run: "run-1" }),
          selectedRunId: "run-1" as string | null,
        },
      },
    );

    rerender({
      isRunsMode: false,
      searchParams: makeSearchParams({ run: "run-1" }),
      selectedRunId: "run-1",
    });

    rerender({
      isRunsMode: true,
      searchParams: makeSearchParams(),
      selectedRunId: null,
    });

    expect(result.current.openRunDetailOnMount).toBe(false);
  });

  it("reopens detail when re-entering runs with a run param that was not dismissed", () => {
    const { result, rerender } = renderHook(
      ({ isRunsMode, searchParams, selectedRunId }) => useRunsDetailState(searchParams, isRunsMode, selectedRunId),
      {
        initialProps: {
          isRunsMode: true,
          searchParams: makeSearchParams({ run: "run-1" }),
          selectedRunId: "run-1" as string | null,
        },
      },
    );

    rerender({
      isRunsMode: false,
      searchParams: makeSearchParams({ run: "run-1" }),
      selectedRunId: "run-1",
    });

    rerender({
      isRunsMode: true,
      searchParams: makeSearchParams({ run: "run-2" }),
      selectedRunId: "run-2",
    });

    expect(result.current.openRunDetailOnMount).toBe(true);
  });

  it("keeps detail closed when the user dismissed detail for the run in the URL", () => {
    const { result, rerender } = renderHook(
      ({ isRunsMode, searchParams, selectedRunId }) => useRunsDetailState(searchParams, isRunsMode, selectedRunId),
      {
        initialProps: {
          isRunsMode: true,
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
      isRunsMode: false,
      searchParams: makeSearchParams({ run: "run-1" }),
      selectedRunId: "run-1",
    });

    rerender({
      isRunsMode: true,
      searchParams: makeSearchParams({ run: "run-1" }),
      selectedRunId: "run-1",
    });

    expect(result.current.openRunDetailOnMount).toBe(false);
  });
});
