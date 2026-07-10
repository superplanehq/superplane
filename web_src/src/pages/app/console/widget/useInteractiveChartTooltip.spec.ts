import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { TOOLTIP_INTERACT_GRACE_MS, useInteractiveChartTooltip } from "./useInteractiveChartTooltip";

describe("useInteractiveChartTooltip", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("holds the tooltip open briefly after the chart point deactivates", () => {
    const { result } = renderHook(() => useInteractiveChartTooltip(true));

    act(() => {
      result.current.syncRechartsActive(true, "a");
    });
    expect(result.current.activeProp).toBeUndefined();

    act(() => {
      result.current.syncRechartsActive(false);
    });
    expect(result.current.activeProp).toBe(true);
    expect(result.current.forceContentActive).toBe(true);

    act(() => {
      vi.advanceTimersByTime(TOOLTIP_INTERACT_GRACE_MS);
    });
    expect(result.current.activeProp).toBeUndefined();
    expect(result.current.forceContentActive).toBe(false);
  });

  it("still dismisses after grace when forced active echoes back through Recharts", () => {
    const { result } = renderHook(() => useInteractiveChartTooltip(true));

    act(() => {
      result.current.syncRechartsActive(true, "a");
    });
    act(() => {
      result.current.syncRechartsActive(false);
    });
    expect(result.current.activeProp).toBe(true);

    // Same-point echo from Tooltip active={true} must not clear the grace timer.
    act(() => {
      result.current.syncRechartsActive(true, "a");
    });
    expect(result.current.activeProp).toBe(true);

    act(() => {
      vi.advanceTimersByTime(TOOLTIP_INTERACT_GRACE_MS);
    });
    expect(result.current.activeProp).toBeUndefined();
    expect(result.current.forceContentActive).toBe(false);
  });

  it("re-arms hold after moving onto another point during grace", () => {
    const { result } = renderHook(() => useInteractiveChartTooltip(true));

    act(() => {
      result.current.syncRechartsActive(true, "a");
      result.current.syncRechartsActive(false);
    });
    expect(result.current.activeProp).toBe(true);

    // Same-point echo while forced.
    act(() => {
      result.current.syncRechartsActive(true, "a");
    });
    // Pointer lands on a different bar before grace ends.
    act(() => {
      result.current.syncRechartsActive(true, "b");
    });
    expect(result.current.activeProp).toBeUndefined();
    expect(result.current.forceContentActive).toBe(false);

    act(() => {
      vi.advanceTimersByTime(TOOLTIP_INTERACT_GRACE_MS);
    });

    // Leaving the new point should start a fresh grace hold for CopyButton.
    act(() => {
      result.current.syncRechartsActive(false);
    });
    expect(result.current.activeProp).toBe(true);

    act(() => {
      vi.advanceTimersByTime(TOOLTIP_INTERACT_GRACE_MS);
    });
    expect(result.current.activeProp).toBeUndefined();
  });

  it("re-arms hold when still on a point after grace ends and active is re-synced", () => {
    const { result } = renderHook(() => useInteractiveChartTooltip(true));

    act(() => {
      result.current.syncRechartsActive(true, "a");
      result.current.syncRechartsActive(false);
      result.current.syncRechartsActive(true, "a"); // echo
    });

    act(() => {
      vi.advanceTimersByTime(TOOLTIP_INTERACT_GRACE_MS);
    });
    expect(result.current.activeProp).toBeUndefined();

    // Bridge re-fires natural active after force releases (forceContentActive edge).
    act(() => {
      result.current.syncRechartsActive(true, "a");
    });

    act(() => {
      result.current.syncRechartsActive(false);
    });
    expect(result.current.activeProp).toBe(true);
  });

  it("keeps the tooltip open while the pointer is over it", () => {
    const { result } = renderHook(() => useInteractiveChartTooltip(true));

    act(() => {
      result.current.syncRechartsActive(true, "a");
      result.current.syncRechartsActive(false);
      result.current.onTooltipEnter();
    });
    act(() => {
      vi.advanceTimersByTime(TOOLTIP_INTERACT_GRACE_MS * 2);
    });
    expect(result.current.activeProp).toBe(true);

    act(() => {
      result.current.onTooltipLeave();
    });
    expect(result.current.activeProp).toBeUndefined();
  });
});
