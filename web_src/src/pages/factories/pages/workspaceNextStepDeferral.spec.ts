import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import {
  readDeferredWorkspaceNextStep,
  useWorkspaceNextStepDeferral,
  WORKSPACE_NEXT_STEP_DEFERRAL_STORAGE_KEY,
  writeDeferredWorkspaceNextStep,
} from "./workspaceNextStepDeferral";

vi.mock("./workspaceNextStepTransition", () => ({
  runWorkspaceNextStepTransition: (update: () => void) => {
    update();
  },
}));

describe("workspaceNextStepDeferral", () => {
  afterEach(() => {
    window.localStorage.clear();
  });

  it("reads and writes a deferred step for one workspace", () => {
    expect(readDeferredWorkspaceNextStep("factory-1")).toBeNull();
    writeDeferredWorkspaceNextStep("factory-1", "pr-checks-handler");
    expect(readDeferredWorkspaceNextStep("factory-1")).toBe("pr-checks-handler");
    expect(readDeferredWorkspaceNextStep("factory-2")).toBeNull();
  });

  it("clears a stored step", () => {
    writeDeferredWorkspaceNextStep("factory-1", "pr-checks-handler");
    writeDeferredWorkspaceNextStep("factory-1", null);
    expect(readDeferredWorkspaceNextStep("factory-1")).toBeNull();
    expect(window.localStorage.getItem(WORKSPACE_NEXT_STEP_DEFERRAL_STORAGE_KEY)).toBeNull();
  });

  it("defers and restores through the hook", () => {
    const { result } = renderHook(() => useWorkspaceNextStepDeferral("factory-1"));

    expect(result.current.deferredStepId).toBeNull();
    act(() => {
      result.current.defer("pr-checks-handler");
    });
    expect(result.current.deferredStepId).toBe("pr-checks-handler");
    expect(readDeferredWorkspaceNextStep("factory-1")).toBe("pr-checks-handler");

    act(() => {
      result.current.restore();
    });
    expect(result.current.deferredStepId).toBeNull();
    expect(readDeferredWorkspaceNextStep("factory-1")).toBeNull();
  });

  it("reads a stored deferral on the first render", () => {
    writeDeferredWorkspaceNextStep("factory-1", "pr-checks-handler");
    const { result } = renderHook(() => useWorkspaceNextStepDeferral("factory-1"));
    expect(result.current.deferredStepId).toBe("pr-checks-handler");
  });

  it("loads the stored step when the workspace changes", () => {
    writeDeferredWorkspaceNextStep("factory-2", "pr-checks-handler");
    const { result, rerender } = renderHook(({ factoryId }) => useWorkspaceNextStepDeferral(factoryId), {
      initialProps: { factoryId: "factory-1" },
    });

    expect(result.current.deferredStepId).toBeNull();
    rerender({ factoryId: "factory-2" });
    expect(result.current.deferredStepId).toBe("pr-checks-handler");
  });
});
