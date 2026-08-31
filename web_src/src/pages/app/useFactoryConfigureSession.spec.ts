import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

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
  it("applies the new spec onto the current workflow snapshot", () => {
    const applyLocalWorkflowUpdate = vi.fn();
    const options = baseOptions({ applyLocalWorkflowUpdate });
    const { result } = renderHook(() => useFactoryConfigureSession(options));
    void result;

    const nextSpec = { nodes: [{ id: "new-node" }], edges: [] };
    options.factoryConfigureActionsRef.current?.applyDraftSpec(nextSpec);

    expect(applyLocalWorkflowUpdate).toHaveBeenCalledWith({
      metadata: { id: "canvas-1", name: "Implement" },
      spec: nextSpec,
    });
  });

  it("does nothing without a current workflow snapshot", () => {
    const applyLocalWorkflowUpdate = vi.fn();
    const options = baseOptions({
      applyLocalWorkflowUpdate,
      getCurrentWorkflowSnapshot: vi.fn(() => null),
    });
    renderHook(() => useFactoryConfigureSession(options));

    options.factoryConfigureActionsRef.current?.applyDraftSpec({ nodes: [], edges: [] });

    expect(applyLocalWorkflowUpdate).not.toHaveBeenCalled();
  });

  it("is not exposed when Configure is inactive", () => {
    const options = baseOptions({ factoryConfigure: false });
    renderHook(() => useFactoryConfigureSession(options));

    expect(options.factoryConfigureActionsRef.current).toBeNull();
  });
});
