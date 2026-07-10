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
      result.current.syncRechartsActive(true);
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
      result.current.syncRechartsActive(true);
    });
    act(() => {
      result.current.syncRechartsActive(false);
    });
    expect(result.current.activeProp).toBe(true);

    // Recharts reports active=true again because we set Tooltip active={true}.
    act(() => {
      result.current.syncRechartsActive(true);
    });
    expect(result.current.activeProp).toBe(true);

    act(() => {
      vi.advanceTimersByTime(TOOLTIP_INTERACT_GRACE_MS);
    });
    expect(result.current.activeProp).toBeUndefined();
    expect(result.current.forceContentActive).toBe(false);
  });

  it("keeps the tooltip open while the pointer is over it", () => {
    const { result } = renderHook(() => useInteractiveChartTooltip(true));

    act(() => {
      result.current.syncRechartsActive(true);
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
