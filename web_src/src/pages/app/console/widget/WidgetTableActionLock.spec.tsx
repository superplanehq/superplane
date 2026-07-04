import type { ReactNode } from "react";
import { renderHook, act } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ConsoleContextProvider } from "../ConsoleContextProvider";
import { WidgetTableActionLockProvider } from "./WidgetTableActionLock";
import { useWidgetTableActionLock } from "./WidgetTableActionLockContext";

let mockInFlight = new Set<string>();

vi.mock("./useInFlightTriggers", () => ({
  useInFlightTriggers: () => ({ inFlight: mockInFlight, isLoading: false }),
}));

function wrapper({ children }: { children: ReactNode }) {
  return (
    <ConsoleContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[]} canRunNodes={false}>
      <WidgetTableActionLockProvider triggerNodeIds={["trigger-1"]}>{children}</WidgetTableActionLockProvider>
    </ConsoleContextProvider>
  );
}

describe("WidgetTableActionLock grace timer", () => {
  beforeEach(() => {
    mockInFlight = new Set();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("keeps inFlightRowByTrigger when runs refresh arrives after endSubmission but before grace elapses", () => {
    const { result, rerender } = renderHook(() => useWidgetTableActionLock(), { wrapper });

    act(() => {
      result.current.beginSubmission("trigger-1", "row-1");
      result.current.endSubmission("trigger-1", "row-1", true);
    });

    mockInFlight = new Set(["trigger-1"]);
    rerender();

    act(() => {
      vi.advanceTimersByTime(1500);
    });

    expect(result.current.inFlightRowByTrigger.get("trigger-1")).toBe("row-1");
    expect(result.current.runInFlightIds.has("trigger-1")).toBe(true);
  });

  it("clears inFlightRowByTrigger after grace when the trigger never appears in flight", () => {
    const { result } = renderHook(() => useWidgetTableActionLock(), { wrapper });

    act(() => {
      result.current.beginSubmission("trigger-1", "row-1");
      result.current.endSubmission("trigger-1", "row-1", true);
    });

    act(() => {
      vi.advanceTimersByTime(1500);
    });

    expect(result.current.inFlightRowByTrigger.has("trigger-1")).toBe(false);
  });
});
