import { renderHook, waitFor } from "@testing-library/react";
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

  it("does not cancel lookup for another node when closing a stale sidebar", async () => {
    const openConfigurationSidebar = vi.fn();
    let resolveLookup: ((value: null) => void) | undefined;
    const resolveLatestNodeRunLookupEvent = vi.fn(
      () =>
        new Promise<null>((resolve) => {
          resolveLookup = resolve;
        }),
    );

    const { result } = renderHook(() =>
      useLiveCanvasNodeClick({
        ...baseOptions,
        resolveLatestNodeRunLookupEvent,
      }),
    );

    result.current.handleLiveCanvasNodeClick("node-b", { openConfigurationSidebar });
    result.current.cancelLiveNodeClickLookup("node-a");

    resolveLookup?.(null);

    await waitFor(() => {
      expect(openConfigurationSidebar).toHaveBeenCalledWith();
    });
  });

  it("cancels lookup when closing the sidebar for the same node", async () => {
    const openConfigurationSidebar = vi.fn();
    let resolveLookup: ((value: null) => void) | undefined;
    const resolveLatestNodeRunLookupEvent = vi.fn(
      () =>
        new Promise<null>((resolve) => {
          resolveLookup = resolve;
        }),
    );

    const { result } = renderHook(() =>
      useLiveCanvasNodeClick({
        ...baseOptions,
        resolveLatestNodeRunLookupEvent,
      }),
    );

    result.current.handleLiveCanvasNodeClick("node-b", { openConfigurationSidebar });
    result.current.cancelLiveNodeClickLookup("node-b");

    resolveLookup?.(null);

    await waitFor(() => {
      expect(resolveLatestNodeRunLookupEvent).toHaveBeenCalled();
    });

    expect(openConfigurationSidebar).not.toHaveBeenCalled();
  });
});
