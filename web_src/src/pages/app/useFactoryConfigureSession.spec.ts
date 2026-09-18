import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

import type { CanvasesCanvas } from "@/api-client";

import { useFactoryConfigureSession, type FactoryConfigureActions } from "./useFactoryConfigureSession";

function baseOptions(overrides: Partial<Parameters<typeof useFactoryConfigureSession>[0]> = {}) {
  return {
    factoryConfigure: true,
    factoryConfigureActionsRef: { current: null } as { current: FactoryConfigureActions | null },
    editSessionActive: true,
    setEditSessionActive: vi.fn(),
    canStageCanvasVersion: true,
    canvasLoading: false,
    liveCanvasVersionLoading: false,
    liveCanvasVersionId: "version-live",
    previewingCurrentVersionRef: { current: false },
    activateCanvasVersionForEditing: vi.fn(),
    draftCanvasSpecsRef: { current: new Map() },
    setDraftCanvasSpec: vi.fn(),
    resyncStagedEditorState: vi.fn().mockResolvedValue(undefined),
    setLastSavedWorkflowSnapshot: vi.fn(),
    commitStagingPending: false,
    resetStagingPending: false,
    activeCanvasVersionIdRef: { current: "version-live" },
    activeCanvasVersionId: "version-live",
    getCurrentWorkflowSnapshot: vi.fn<() => CanvasesCanvas | null | undefined>(() => ({
      metadata: { id: "canvas-1", name: "Implement" },
      spec: { nodes: [{ id: "old-node" }], edges: [] },
    })),
    updateCanvasVersionMutation: { mutateAsync: vi.fn() },
    handleCommitStaging: vi.fn(),
    handleResetStaging: vi.fn(),
    handleExitEditSession: vi.fn(),
    hasStagingChanges: false,
    hasUncommittedCanvasDraftChanges: false,
    applyLocalWorkflowUpdate: vi.fn(),
    ...overrides,
  };
}

describe("useFactoryConfigureSession applyDraftSpec", () => {
  it("applies the new spec onto the current workflow snapshot", async () => {
    const applyLocalWorkflowUpdate = vi.fn();
    const options = baseOptions({ applyLocalWorkflowUpdate });
    const { result } = renderHook(() => useFactoryConfigureSession(options));
    void result;

    const nextSpec = { nodes: [{ id: "new-node" }], edges: [] };
    await options.factoryConfigureActionsRef.current?.applyDraftSpec(nextSpec);

    expect(applyLocalWorkflowUpdate).toHaveBeenCalledWith({
      metadata: { id: "canvas-1", name: "Implement" },
      spec: nextSpec,
    });
  });

  it("lays out the merged workflow before applying it", async () => {
    const applyLocalWorkflowUpdate = vi.fn();
    const laidOut = {
      metadata: { id: "canvas-1", name: "Implement" },
      spec: { nodes: [{ id: "laid-out" }], edges: [] },
    };
    const layoutDraftWorkflow = vi.fn(async () => laidOut);
    const options = baseOptions({ applyLocalWorkflowUpdate, layoutDraftWorkflow });
    renderHook(() => useFactoryConfigureSession(options));

    const nextSpec = { nodes: [{ id: "new-node" }], edges: [] };
    await options.factoryConfigureActionsRef.current?.applyDraftSpec(nextSpec);

    expect(layoutDraftWorkflow).toHaveBeenCalledWith({
      metadata: { id: "canvas-1", name: "Implement" },
      spec: nextSpec,
    });
    expect(applyLocalWorkflowUpdate).toHaveBeenCalledWith(laidOut);
  });

  it("does not apply a layout result after the session changes", async () => {
    const applyLocalWorkflowUpdate = vi.fn();
    let releaseLayout: (workflow: CanvasesCanvas) => void = () => {};
    const layoutDraftWorkflow = vi.fn(
      () =>
        new Promise<CanvasesCanvas>((resolve) => {
          releaseLayout = resolve;
        }),
    );
    const options = baseOptions({ applyLocalWorkflowUpdate, layoutDraftWorkflow });
    renderHook(() => useFactoryConfigureSession(options));

    const pending = options.factoryConfigureActionsRef.current?.applyDraftSpec({
      nodes: [{ id: "new-node" }],
      edges: [],
    });
    options.activeCanvasVersionIdRef.current = "version-other";
    releaseLayout({
      metadata: { id: "canvas-1", name: "Implement" },
      spec: { nodes: [{ id: "laid-out" }], edges: [] },
    });
    await pending;

    expect(applyLocalWorkflowUpdate).not.toHaveBeenCalled();
  });

  it("does nothing without a current workflow snapshot", async () => {
    const applyLocalWorkflowUpdate = vi.fn();
    const options = baseOptions({
      applyLocalWorkflowUpdate,
      getCurrentWorkflowSnapshot: vi.fn(() => null),
    });
    renderHook(() => useFactoryConfigureSession(options));

    await options.factoryConfigureActionsRef.current?.applyDraftSpec({ nodes: [], edges: [] });

    expect(applyLocalWorkflowUpdate).not.toHaveBeenCalled();
  });

  it("is not exposed when Configure is inactive", () => {
    const options = baseOptions({ factoryConfigure: false });
    renderHook(() => useFactoryConfigureSession(options));

    expect(options.factoryConfigureActionsRef.current).toBeNull();
  });
});
