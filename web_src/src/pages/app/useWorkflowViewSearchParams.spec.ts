import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useWorkflowViewSearchParams } from "./useWorkflowViewSearchParams";

function makeSearchParams(params: Record<string, string> = {}) {
  return new URLSearchParams(params);
}

describe("useWorkflowViewSearchParams", () => {
  it("derives run inspection mode from the run param on the same render", () => {
    const setSearchParams = vi.fn();
    const { result, rerender } = renderHook(
      ({ searchParams }) => useWorkflowViewSearchParams(searchParams, setSearchParams),
      { initialProps: { searchParams: makeSearchParams() } },
    );

    expect(result.current.isRunInspectionMode).toBe(false);

    rerender({ searchParams: makeSearchParams({ run: "run-42" }) });

    expect(result.current.isRunInspectionMode).toBe(true);
    expect(result.current.selectedRunId).toBe("run-42");
  });

  it("migrates legacy runs view params", async () => {
    const setSearchParams = vi.fn();
    renderHook(() => useWorkflowViewSearchParams(makeSearchParams({ view: "runs", run: "run-42" }), setSearchParams));

    await waitFor(() => expect(setSearchParams).toHaveBeenCalled());

    const next = setSearchParams.mock.calls[0]?.[0] as URLSearchParams;

    expect(next.get("view")).toBeNull();
    expect(next.get("run")).toBe("run-42");
  });

  it("migrates legacy versions view params", async () => {
    const setSearchParams = vi.fn();
    renderHook(() => useWorkflowViewSearchParams(makeSearchParams({ view: "versions" }), setSearchParams));

    await waitFor(() => expect(setSearchParams).toHaveBeenCalled());

    const next = setSearchParams.mock.calls[0]?.[0] as URLSearchParams;

    expect(next.get("view")).toBeNull();
  });

  it("migrates conflicting non-canvas views when run inspection is requested", async () => {
    const setSearchParams = vi.fn();
    renderHook(() =>
      useWorkflowViewSearchParams(makeSearchParams({ view: "console", run: "run-42" }), setSearchParams),
    );

    await waitFor(() => expect(setSearchParams).toHaveBeenCalled());

    const next = setSearchParams.mock.calls[0]?.[0] as URLSearchParams;

    expect(next.get("view")).toBeNull();
    expect(next.get("run")).toBe("run-42");
  });

  it("migrates conflicting run params on non-canvas views", async () => {
    const setSearchParams = vi.fn();
    renderHook(() => useWorkflowViewSearchParams(makeSearchParams({ view: "memory", run: "run-42" }), setSearchParams));

    await waitFor(() => expect(setSearchParams).toHaveBeenCalled());

    const next = setSearchParams.mock.calls[0]?.[0] as URLSearchParams;

    expect(next.get("view")).toBeNull();
    expect(next.get("run")).toBe("run-42");
  });

  it("accepts legacy dashboard view links and migrates them to console", async () => {
    const setSearchParams = vi.fn();
    const { result } = renderHook(() =>
      useWorkflowViewSearchParams(makeSearchParams({ view: "dashboard" }), setSearchParams),
    );

    expect(result.current.isConsoleMode).toBe(true);

    await waitFor(() => expect(setSearchParams).toHaveBeenCalled());

    const next = setSearchParams.mock.calls[0]?.[0] as URLSearchParams;

    expect(next.get("view")).toBe("console");
  });
});
