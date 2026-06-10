import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useWorkflowViewSearchParams } from "./useWorkflowViewSearchParams";

function makeSearchParams(params: Record<string, string> = {}) {
  return new URLSearchParams(params);
}

describe("useWorkflowViewSearchParams", () => {
  it("derives runs and versions mode from the URL on the same render", () => {
    const setSearchParams = vi.fn();
    const { result, rerender } = renderHook(
      ({ searchParams }) => useWorkflowViewSearchParams(searchParams, setSearchParams),
      { initialProps: { searchParams: makeSearchParams() } },
    );

    expect(result.current.isRunsMode).toBe(false);
    expect(result.current.isVersionsMode).toBe(false);

    rerender({ searchParams: makeSearchParams({ view: "runs" }) });

    expect(result.current.isRunsMode).toBe(true);
    expect(result.current.isVersionsMode).toBe(false);

    rerender({ searchParams: makeSearchParams({ view: "versions" }) });

    expect(result.current.isRunsMode).toBe(false);
    expect(result.current.isVersionsMode).toBe(true);
  });

  it("derives selectedRunId from the run search param on the same render", () => {
    const setSearchParams = vi.fn();
    const { result, rerender } = renderHook(
      ({ searchParams }) => useWorkflowViewSearchParams(searchParams, setSearchParams),
      { initialProps: { searchParams: makeSearchParams({ view: "runs" }) } },
    );

    expect(result.current.selectedRunId).toBeNull();

    rerender({ searchParams: makeSearchParams({ view: "runs", run: "run-42" }) });

    expect(result.current.selectedRunId).toBe("run-42");
  });

  it("accepts legacy dashboard view links and migrates them to console", () => {
    const setSearchParams = vi.fn();
    const { result } = renderHook(() =>
      useWorkflowViewSearchParams(makeSearchParams({ view: "dashboard" }), setSearchParams),
    );

    expect(result.current.isConsoleMode).toBe(true);
    expect(setSearchParams).toHaveBeenCalledTimes(1);

    const updater = setSearchParams.mock.calls[0]?.[0] as (current: URLSearchParams) => URLSearchParams;
    const next = updater(makeSearchParams({ view: "dashboard", run: "run-42" }));

    expect(next.get("view")).toBe("console");
    expect(next.get("run")).toBe("run-42");
  });
});
