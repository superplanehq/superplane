import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useStaleRunInspectionUrlCleanup } from "./useStaleRunInspectionUrlCleanup";

describe("useStaleRunInspectionUrlCleanup", () => {
  it("clears immediately for malformed run ids", () => {
    const onClear = vi.fn();

    renderHook(() =>
      useStaleRunInspectionUrlCleanup({
        selectedRunId: "not-a-uuid",
        isRunInspectionMode: true,
        selectedRun: null,
        isRunResolveLoading: false,
        describeRunSettled: true,
        onClear,
      }),
    );

    expect(onClear).toHaveBeenCalledTimes(1);
  });

  it("clears after describe settles without resolving the run", () => {
    const onClear = vi.fn();

    renderHook(() =>
      useStaleRunInspectionUrlCleanup({
        selectedRunId: "550e8400-e29b-41d4-a716-446655440000",
        isRunInspectionMode: true,
        selectedRun: null,
        isRunResolveLoading: false,
        describeRunSettled: true,
        onClear,
      }),
    );

    expect(onClear).toHaveBeenCalledTimes(1);
  });

  it("waits for describe to settle before clearing", () => {
    const onClear = vi.fn();

    renderHook(() =>
      useStaleRunInspectionUrlCleanup({
        selectedRunId: "550e8400-e29b-41d4-a716-446655440000",
        isRunInspectionMode: true,
        selectedRun: null,
        isRunResolveLoading: true,
        describeRunSettled: false,
        onClear,
      }),
    );

    expect(onClear).not.toHaveBeenCalled();
  });

  it("does not clear while the run is still loading", () => {
    const onClear = vi.fn();

    renderHook(() =>
      useStaleRunInspectionUrlCleanup({
        selectedRunId: "550e8400-e29b-41d4-a716-446655440000",
        isRunInspectionMode: true,
        selectedRun: null,
        isRunResolveLoading: true,
        describeRunSettled: false,
        onClear,
      }),
    );

    expect(onClear).not.toHaveBeenCalled();
  });

  it("does not clear when the run resolved", () => {
    const onClear = vi.fn();

    renderHook(() =>
      useStaleRunInspectionUrlCleanup({
        selectedRunId: "550e8400-e29b-41d4-a716-446655440000",
        isRunInspectionMode: true,
        selectedRun: { id: "550e8400-e29b-41d4-a716-446655440000" },
        isRunResolveLoading: false,
        describeRunSettled: true,
        onClear,
      }),
    );

    expect(onClear).not.toHaveBeenCalled();
  });
});
