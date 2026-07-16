import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useLiveCanvasNodeClick } from "./useLiveCanvasNodeClick";

describe("useLiveCanvasNodeClick", () => {
  const baseOptions = {
    canvasNodesById: new Map(),
    fetchRunIdForSidebarEvent: vi.fn(),
    handleSelectRunFromSidebarEvent: vi.fn(),
    isEditing: false,
    isRunInspectionMode: false,
    liveSidebarRunLookupEnabled: true,
    resolveLatestNodeRunLookupEvent: vi.fn(),
    resolveRunIdForSidebarEvent: vi.fn(),
  };

  it("opens the configuration sidebar when live run lookup is disabled", () => {
    const openConfigurationSidebar = vi.fn();
    const { result } = renderHook(() =>
      useLiveCanvasNodeClick({
        ...baseOptions,
        liveSidebarRunLookupEnabled: false,
      }),
    );

    result.current.handleLiveCanvasNodeClick("node-1", { openConfigurationSidebar });

    expect(openConfigurationSidebar).toHaveBeenCalledWith();
  });

  it("does nothing while editing", () => {
    const openConfigurationSidebar = vi.fn();
    const { result } = renderHook(() =>
      useLiveCanvasNodeClick({
        ...baseOptions,
        isEditing: true,
        liveSidebarRunLookupEnabled: false,
      }),
    );

    result.current.handleLiveCanvasNodeClick("node-1", { openConfigurationSidebar });

    expect(openConfigurationSidebar).not.toHaveBeenCalled();
  });
});
