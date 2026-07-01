import { act, renderHook, waitFor } from "@testing-library/react";
import type { SetURLSearchParams } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { useWorkflowHeaderEditActions } from "./useWorkflowHeaderEditActions";

function renderWorkflowHeaderEditActions(overrides: Partial<Parameters<typeof useWorkflowHeaderEditActions>[0]> = {}) {
  const config = {
    isRunInspectionMode: false,
    handleClearRunInspection: vi.fn(),
    handleToggleEditMode: vi.fn().mockResolvedValue(true),
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
    const { result, config } = renderWorkflowHeaderEditActions({ isRunInspectionMode: true });

    await act(async () => {
      await result.current.handleEnterEditModeFromHeader();
    });

    expect(config.setRunDetailNodeId).toHaveBeenCalledWith(null);
    expect(config.setSearchParams).toHaveBeenCalledTimes(1);
    expect(config.handleToggleEditMode).toHaveBeenCalledTimes(1);
  });

  it("still exits edit mode when a run is in the URL", async () => {
    const { result, config } = renderWorkflowHeaderEditActions({ isRunInspectionMode: true });

    await act(async () => {
      await result.current.handleExitEditModeFromHeader();
    });

    expect(config.handleClearRunInspection).toHaveBeenCalledTimes(1);
    expect(config.handleToggleEditMode).toHaveBeenCalledTimes(1);
  });

  it("auto edit mode removes only the edit param from the current URL", async () => {
    const handleToggleEditMode = vi.fn().mockResolvedValue(true);
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
          editEntryReady: true,
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
    const handleToggleEditMode = vi.fn().mockResolvedValue(true);
    const setSearchParams = vi.fn();
    const setRunDetailNodeId = vi.fn();
    const searchParams = new URLSearchParams("edit=1&run=run-123");
    const callOrder: string[] = [];

    setSearchParams.mockImplementation(() => {
      callOrder.push("setSearchParams");
    });
    handleToggleEditMode.mockImplementation(async () => {
      callOrder.push("toggleEditMode");
      return true;
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
          editEntryReady: true,
        },
      }),
    );

    await waitFor(() => {
      expect(handleToggleEditMode).toHaveBeenCalledTimes(1);
    });

    expect(setRunDetailNodeId).toHaveBeenCalledWith(null);
    expect(setSearchParams).toHaveBeenCalledTimes(2);
    expect(callOrder).toEqual(["setSearchParams", "toggleEditMode", "setSearchParams"]);

    const clearRunUpdater = setSearchParams.mock.calls[0]?.[0] as (current: URLSearchParams) => URLSearchParams;
    const clearedRun = clearRunUpdater(new URLSearchParams("edit=1&run=run-123"));
    expect(clearedRun.get("run")).toBeNull();
    expect(clearedRun.get("edit")).toBe("1");

    const clearEditUpdater = setSearchParams.mock.calls[1]?.[0] as (current: URLSearchParams) => URLSearchParams;
    const clearedEdit = clearEditUpdater(new URLSearchParams("edit=1"));
    expect(clearedEdit.get("edit")).toBeNull();
  });

  it("waits for edit entry prerequisites before auto edit mode runs", async () => {
    const handleToggleEditMode = vi.fn().mockResolvedValue(true);
    const setSearchParams = vi.fn();
    const searchParams = new URLSearchParams("edit=1");

    const { rerender } = renderHook(
      ({ editEntryReady }) =>
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
            editEntryReady,
          },
        }),
      { initialProps: { editEntryReady: false } },
    );

    await act(async () => {
      await Promise.resolve();
    });
    expect(handleToggleEditMode).not.toHaveBeenCalled();

    rerender({ editEntryReady: true });

    await waitFor(() => {
      expect(handleToggleEditMode).toHaveBeenCalledTimes(1);
    });
  });

  it("does not re-trigger auto edit when search params change during toggle", async () => {
    let resolveToggle!: (value: boolean) => void;
    const togglePromise = new Promise<boolean>((resolve) => {
      resolveToggle = resolve;
    });
    const handleToggleEditMode = vi.fn().mockReturnValue(togglePromise);
    const setSearchParams = vi.fn();

    const { rerender } = renderHook(
      ({ searchParams }) =>
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
            editEntryReady: true,
          },
        }),
      { initialProps: { searchParams: new URLSearchParams("edit=1") } },
    );

    await act(async () => {
      await Promise.resolve();
    });
    expect(handleToggleEditMode).toHaveBeenCalledTimes(1);

    rerender({ searchParams: new URLSearchParams("edit=1&branch=main") });

    await act(async () => {
      await Promise.resolve();
    });
    expect(handleToggleEditMode).toHaveBeenCalledTimes(1);

    await act(async () => {
      resolveToggle(true);
      await togglePromise;
    });
  });

  it("does not consume edit=1 when entering edit fails", async () => {
    const handleToggleEditMode = vi.fn().mockResolvedValue(false);
    const setSearchParams = vi.fn();
    const searchParams = new URLSearchParams("edit=1");

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
          editEntryReady: true,
        },
      }),
    );

    await waitFor(() => {
      expect(handleToggleEditMode).toHaveBeenCalledTimes(1);
    });
    expect(setSearchParams).not.toHaveBeenCalled();
  });
});
