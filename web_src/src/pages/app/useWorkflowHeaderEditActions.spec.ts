import { act, renderHook, waitFor } from "@testing-library/react";
import type { SetURLSearchParams } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { useWorkflowHeaderEditActions } from "./useWorkflowHeaderEditActions";

function renderWorkflowHeaderEditActions(overrides: Partial<Parameters<typeof useWorkflowHeaderEditActions>[0]> = {}) {
  const config = {
    isRunInspectionMode: false,
    handleClearRunInspection: vi.fn(),
    handleToggleEditMode: vi.fn().mockResolvedValue(undefined),
    setRunDetailNodeId: vi.fn(),
    setSearchParams: vi.fn() as unknown as SetURLSearchParams,
    ...overrides,
  };

  const result = renderHook(() => useWorkflowHeaderEditActions(config));

  return { ...result, config };
}

describe("useWorkflowHeaderEditActions", () => {
  it("does not clear the current view when entering edit mode outside run inspection", async () => {
    const { result, config } = renderWorkflowHeaderEditActions();

    await act(async () => {
      await result.current.handleEnterEditModeFromHeader();
    });

    expect(config.handleToggleEditMode).toHaveBeenCalledTimes(1);
    expect(config.setSearchParams).not.toHaveBeenCalled();
  });

  it("does not clear the current view when exiting edit mode outside run inspection", async () => {
    const { result, config } = renderWorkflowHeaderEditActions();

    await act(async () => {
      await result.current.handleExitEditModeFromHeader();
    });

    expect(config.handleToggleEditMode).toHaveBeenCalledTimes(1);
    expect(config.setSearchParams).not.toHaveBeenCalled();
  });

  it("clears run inspection before entering edit mode", async () => {
    const callOrder: string[] = [];
    const handleClearRunInspection = vi.fn(() => {
      callOrder.push("clearRunInspection");
    });
    const handleToggleEditMode = vi.fn(async () => {
      callOrder.push("toggleEditMode");
    });
    const { result, config } = renderWorkflowHeaderEditActions({
      isRunInspectionMode: true,
      handleClearRunInspection,
      handleToggleEditMode,
    });

    await act(async () => {
      await result.current.handleEnterEditModeFromHeader();
    });

    expect(config.handleClearRunInspection).toHaveBeenCalledTimes(2);
    expect(config.setRunDetailNodeId).not.toHaveBeenCalled();
    expect(config.setSearchParams).not.toHaveBeenCalled();
    expect(config.handleToggleEditMode).toHaveBeenCalledTimes(1);
    expect(callOrder).toEqual(["clearRunInspection", "toggleEditMode", "clearRunInspection"]);
  });

  it("still exits edit mode when a run is in the URL", async () => {
    const { result, config } = renderWorkflowHeaderEditActions({ isRunInspectionMode: true });

    await act(async () => {
      await result.current.handleExitEditModeFromHeader();
    });

    expect(config.handleClearRunInspection).toHaveBeenCalledTimes(1);
    expect(config.handleToggleEditMode).toHaveBeenCalledTimes(1);
  });

  it("clears run inspection when an edit session exit needs to leave inspection mode", () => {
    const { result, config } = renderWorkflowHeaderEditActions({ isRunInspectionMode: true });

    act(() => {
      result.current.clearRunInspectionForEdit();
    });

    expect(config.handleClearRunInspection).toHaveBeenCalledTimes(1);
    expect(config.setRunDetailNodeId).not.toHaveBeenCalled();
    expect(config.setSearchParams).not.toHaveBeenCalled();
  });

  it("auto edit mode waits for live version data before entering edit mode", async () => {
    const handleToggleEditMode = vi.fn().mockResolvedValue(undefined);
    const setSearchParams = vi.fn();
    const searchParams = new URLSearchParams("edit=1");

    const { rerender } = renderHook(
      ({ liveVersionLoading }) =>
        useWorkflowHeaderEditActions({
          isRunInspectionMode: false,
          handleClearRunInspection: vi.fn(),
          handleToggleEditMode,
          setRunDetailNodeId: vi.fn(),
          setSearchParams: setSearchParams as unknown as SetURLSearchParams,
          startup: {
            hasEditableVersion: false,
            canUpdateCanvas: true,
            canvas: { metadata: { id: "canvas-1" }, spec: {} },
            liveVersionLoading,
            searchParams,
          },
        }),
      { initialProps: { liveVersionLoading: true } },
    );

    expect(handleToggleEditMode).not.toHaveBeenCalled();

    rerender({ liveVersionLoading: false });

    await waitFor(() => {
      expect(handleToggleEditMode).toHaveBeenCalledTimes(1);
    });
  });

  it("auto edit mode removes only the edit param from the current URL", async () => {
    const handleToggleEditMode = vi.fn().mockResolvedValue(undefined);
    const setSearchParams = vi.fn();
    const searchParams = new URLSearchParams("edit=1&version=draft-version&branch=drafts%2Fabc");

    renderHook(() =>
      useWorkflowHeaderEditActions({
        isRunInspectionMode: false,
        handleClearRunInspection: vi.fn(),
        handleToggleEditMode,
        setRunDetailNodeId: vi.fn(),
        setSearchParams: setSearchParams as unknown as SetURLSearchParams,
        startup: {
          hasEditableVersion: false,
          canUpdateCanvas: true,
          canvas: { metadata: { id: "canvas-1" }, spec: {} },
          searchParams,
        },
      }),
    );

    await waitFor(() => {
      expect(handleToggleEditMode).toHaveBeenCalledTimes(1);
    });

    const updater = setSearchParams.mock.calls[0]?.[0] as (current: URLSearchParams) => URLSearchParams;
    const next = updater(new URLSearchParams("edit=1&version=draft-version&branch=drafts%2Fabc"));

    expect(next.get("edit")).toBeNull();
    expect(next.get("version")).toBe("draft-version");
    expect(next.get("branch")).toBe("drafts/abc");
  });

  it("auto edit mode clears run inspection before entering edit mode", async () => {
    const handleToggleEditMode = vi.fn().mockResolvedValue(undefined);
    const setSearchParams = vi.fn();
    const setRunDetailNodeId = vi.fn();
    const searchParams = new URLSearchParams("edit=1&run=run-123&sidebar=runs&node=node-1");
    const callOrder: string[] = [];

    setSearchParams.mockImplementation(() => {
      callOrder.push("setSearchParams");
    });
    handleToggleEditMode.mockImplementation(async () => {
      callOrder.push("toggleEditMode");
    });

    renderHook(() =>
      useWorkflowHeaderEditActions({
        isRunInspectionMode: true,
        handleClearRunInspection: vi.fn(),
        handleToggleEditMode,
        setRunDetailNodeId,
        setSearchParams: setSearchParams as unknown as SetURLSearchParams,
        startup: {
          hasEditableVersion: false,
          canUpdateCanvas: true,
          canvas: { metadata: { id: "canvas-1" }, spec: {} },
          searchParams,
        },
      }),
    );

    await waitFor(() => {
      expect(handleToggleEditMode).toHaveBeenCalledTimes(1);
    });

    expect(setRunDetailNodeId).toHaveBeenCalledWith(null);
    expect(setSearchParams).toHaveBeenCalledTimes(3);
    expect(callOrder).toEqual(["setSearchParams", "toggleEditMode", "setSearchParams", "setSearchParams"]);

    const clearRunUpdater = setSearchParams.mock.calls[0]?.[0] as (current: URLSearchParams) => URLSearchParams;
    const clearedRun = clearRunUpdater(new URLSearchParams("edit=1&run=run-123&sidebar=runs&node=node-1"));
    expect(clearedRun.get("run")).toBeNull();
    expect(clearedRun.get("sidebar")).toBeNull();
    expect(clearedRun.get("node")).toBeNull();
    expect(clearedRun.get("edit")).toBe("1");

    const clearRunAfterEditUpdater = setSearchParams.mock.calls[1]?.[0] as (
      current: URLSearchParams,
    ) => URLSearchParams;
    const clearedRunAfterEdit = clearRunAfterEditUpdater(
      new URLSearchParams("edit=1&run=run-123&sidebar=runs&node=node-1"),
    );
    expect(clearedRunAfterEdit.get("run")).toBeNull();
    expect(clearedRunAfterEdit.get("sidebar")).toBeNull();
    expect(clearedRunAfterEdit.get("node")).toBeNull();
    expect(clearedRunAfterEdit.get("edit")).toBe("1");

    const clearEditUpdater = setSearchParams.mock.calls[2]?.[0] as (current: URLSearchParams) => URLSearchParams;
    const clearedEdit = clearEditUpdater(new URLSearchParams("edit=1"));
    expect(clearedEdit.get("edit")).toBeNull();
  });
});
