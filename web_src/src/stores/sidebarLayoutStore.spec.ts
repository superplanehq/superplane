import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import {
  useEffectiveLeftSidebarWidth,
  useEffectiveRightSidebarWidth,
  useSidebarLayoutStore,
} from "./sidebarLayoutStore";

describe("sidebar layout effective widths", () => {
  beforeEach(() => {
    localStorage.clear();
    useSidebarLayoutStore.getState().hydrateFromStorage();
  });

  it("returns 0 right inset when no right sidebar is mounted", () => {
    useSidebarLayoutStore.setState({ rightWidth: 420, rightMountCount: 0 });

    const { result } = renderHook(() => useEffectiveRightSidebarWidth());

    expect(result.current).toBe(0);
  });

  it("returns rightWidth when a right sidebar is mounted", () => {
    useSidebarLayoutStore.setState({ rightWidth: 420, rightMountCount: 1 });

    const { result } = renderHook(() => useEffectiveRightSidebarWidth());

    expect(result.current).toBe(420);
  });

  it("sums left and aux left widths when both are mounted", () => {
    useSidebarLayoutStore.setState({
      leftWidth: 380,
      auxLeftWidth: 240,
      leftMountCount: 1,
      auxLeftMountCount: 1,
    });

    const { result } = renderHook(() => useEffectiveLeftSidebarWidth());

    expect(result.current).toBe(620);
  });
});
