import { act, renderHook } from "@testing-library/react";
import { describe, it, expect } from "vitest";

import { INITIAL_EAGER_ROWS, LOAD_MORE_STEP } from "./widgetPagination";
import { useDisplayWindow } from "./useWidgetData";

describe("useDisplayWindow — non-progressive callers", () => {
  it("renders the full effective limit and tracks live limit increases", () => {
    const { result, rerender } = renderHook(
      ({ effectiveLimit }) => useDisplayWindow({ dataSourceKind: "runs", progressive: false, effectiveLimit }),
      { initialProps: { effectiveLimit: 50 } },
    );

    expect(result.current.displaySlice).toBe(50);

    // Regression guard: raising the limit live must take effect immediately
    // (no remount), not stay clamped to the stale initial display count.
    rerender({ effectiveLimit: 200 });
    expect(result.current.displaySlice).toBe(200);

    rerender({ effectiveLimit: 25 });
    expect(result.current.displaySlice).toBe(25);
  });

  it("ignores loadMore (no progressive window)", () => {
    const { result } = renderHook(() =>
      useDisplayWindow({ dataSourceKind: "runs", progressive: false, effectiveLimit: 100 }),
    );

    act(() => result.current.loadMore());
    expect(result.current.displaySlice).toBe(100);
  });
});

describe("useDisplayWindow — progressive callers", () => {
  it("starts at the eager window and grows by LOAD_MORE_STEP up to a finite limit", () => {
    const { result } = renderHook(() =>
      useDisplayWindow({ dataSourceKind: "runs", progressive: true, effectiveLimit: INITIAL_EAGER_ROWS + 150 }),
    );

    expect(result.current.displayCount).toBe(INITIAL_EAGER_ROWS);

    act(() => result.current.loadMore());
    expect(result.current.displayCount).toBe(INITIAL_EAGER_ROWS + LOAD_MORE_STEP);

    // Caps at the configured limit rather than overshooting.
    act(() => result.current.loadMore());
    expect(result.current.displayCount).toBe(INITIAL_EAGER_ROWS + 150);
  });

  it("does not reset the window when only the limit changes", () => {
    const { result, rerender } = renderHook(
      ({ effectiveLimit }) => useDisplayWindow({ dataSourceKind: "runs", progressive: true, effectiveLimit }),
      { initialProps: { effectiveLimit: Number.POSITIVE_INFINITY } },
    );

    act(() => result.current.loadMore());
    expect(result.current.displayCount).toBe(INITIAL_EAGER_ROWS + LOAD_MORE_STEP);

    rerender({ effectiveLimit: 1000 });
    expect(result.current.displayCount).toBe(INITIAL_EAGER_ROWS + LOAD_MORE_STEP);
  });

  it("resets the window when the data source kind changes", () => {
    const { result, rerender } = renderHook(
      ({ dataSourceKind }) =>
        useDisplayWindow({ dataSourceKind, progressive: true, effectiveLimit: Number.POSITIVE_INFINITY }),
      { initialProps: { dataSourceKind: "runs" as "runs" | "executions" } },
    );

    act(() => result.current.loadMore());
    expect(result.current.displayCount).toBe(INITIAL_EAGER_ROWS + LOAD_MORE_STEP);

    rerender({ dataSourceKind: "executions" });
    expect(result.current.displayCount).toBe(INITIAL_EAGER_ROWS);
  });
});
