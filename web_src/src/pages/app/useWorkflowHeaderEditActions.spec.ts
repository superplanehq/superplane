import { act, renderHook } from "@testing-library/react";
import type { SetURLSearchParams } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { useWorkflowHeaderEditActions } from "./useWorkflowHeaderEditActions";

function renderWorkflowHeaderEditActions(overrides: Partial<Parameters<typeof useWorkflowHeaderEditActions>[0]> = {}) {
  const config = {
    isRunsMode: false,
    handleExitRunsMode: vi.fn(),
    handleToggleEditMode: vi.fn().mockResolvedValue(undefined),
    setIsRunsMode: vi.fn(),
    setSelectedRunId: vi.fn(),
    setRunDetailNodeId: vi.fn(),
    setSearchParams: vi.fn() as unknown as SetURLSearchParams,
    ...overrides,
  };

  const result = renderHook(() => useWorkflowHeaderEditActions(config));

  return { ...result, config };
}

describe("useWorkflowHeaderEditActions", () => {
  it("does not clear the current view when entering edit mode outside runs", async () => {
    const { result, config } = renderWorkflowHeaderEditActions();

    await act(async () => {
      await result.current.handleEnterEditModeFromHeader();
    });

    expect(config.handleToggleEditMode).toHaveBeenCalledTimes(1);
    expect(config.setSearchParams).not.toHaveBeenCalled();
  });

  it("does not clear the current view when exiting edit mode outside runs", async () => {
    const { result, config } = renderWorkflowHeaderEditActions();

    await act(async () => {
      await result.current.handleExitEditModeFromHeader();
    });

    expect(config.handleToggleEditMode).toHaveBeenCalledTimes(1);
    expect(config.setSearchParams).not.toHaveBeenCalled();
  });

  it("clears runs view before entering edit mode", async () => {
    const { result, config } = renderWorkflowHeaderEditActions({ isRunsMode: true });

    await act(async () => {
      await result.current.handleEnterEditModeFromHeader();
    });

    expect(config.setIsRunsMode).toHaveBeenCalledWith(false);
    expect(config.setSelectedRunId).toHaveBeenCalledWith(null);
    expect(config.setRunDetailNodeId).toHaveBeenCalledWith(null);
    expect(config.setSearchParams).toHaveBeenCalledTimes(1);
    expect(config.handleToggleEditMode).toHaveBeenCalledTimes(1);
  });
});
