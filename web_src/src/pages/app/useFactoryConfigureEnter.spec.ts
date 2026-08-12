import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { useFactoryConfigureEnter } from "./useFactoryConfigureEnter";

describe("useFactoryConfigureEnter", () => {
  it("does not re-enter Configure when activateCanvasVersionForEditing identity changes before edit is enabled", async () => {
    const activateCanvasVersionForEditing = vi.fn();
    const setDraftCanvasSpec = vi.fn();
    const setEditSessionActive = vi.fn();
    const resyncStagedEditorState = vi.fn(() => new Promise<void>(() => {}));

    const { rerender } = renderHook(
      ({ activate }: { activate: typeof activateCanvasVersionForEditing }) =>
        useFactoryConfigureEnter({
          factoryConfigure: true,
          editSessionActive: false,
          setEditSessionActive,
          canStageCanvasVersion: true,
          canvasLoading: false,
          liveCanvasVersionLoading: false,
          liveCanvasVersionId: "version-live",
          liveCanvasVersion: { metadata: { id: "version-live" }, spec: { nodes: [], edges: [] } },
          liveCanvas: { metadata: { id: "canvas-1", liveVersionId: "version-live" }, spec: { nodes: [], edges: [] } },
          previewingCurrentVersionRef: { current: false },
          activateCanvasVersionForEditing: activate,
          draftCanvasSpecsRef: { current: new Map() },
          setDraftCanvasSpec,
          resyncStagedEditorState,
          setLastSavedWorkflowSnapshot: vi.fn(),
        }),
      { initialProps: { activate: activateCanvasVersionForEditing } },
    );

    expect(activateCanvasVersionForEditing).toHaveBeenCalledTimes(1);

    const nextActivate = vi.fn();
    await act(async () => {
      rerender({ activate: nextActivate });
    });

    expect(nextActivate).not.toHaveBeenCalled();
    expect(activateCanvasVersionForEditing).toHaveBeenCalledTimes(1);
  });

  it("allows a fresh Configure enter after leaving configure mode", async () => {
    const activateCanvasVersionForEditing = vi.fn();
    const resyncStagedEditorState = vi.fn().mockResolvedValue(undefined);

    const { rerender } = renderHook(
      ({ factoryConfigure }: { factoryConfigure: boolean }) =>
        useFactoryConfigureEnter({
          factoryConfigure,
          editSessionActive: false,
          setEditSessionActive: vi.fn(),
          canStageCanvasVersion: true,
          canvasLoading: false,
          liveCanvasVersionLoading: false,
          liveCanvasVersionId: "version-live",
          liveCanvasVersion: { metadata: { id: "version-live" }, spec: { nodes: [], edges: [] } },
          liveCanvas: { metadata: { id: "canvas-1", liveVersionId: "version-live" }, spec: { nodes: [], edges: [] } },
          previewingCurrentVersionRef: { current: false },
          activateCanvasVersionForEditing,
          draftCanvasSpecsRef: { current: new Map() },
          setDraftCanvasSpec: vi.fn(),
          resyncStagedEditorState,
          setLastSavedWorkflowSnapshot: vi.fn(),
        }),
      { initialProps: { factoryConfigure: true } },
    );

    expect(activateCanvasVersionForEditing).toHaveBeenCalledTimes(1);

    await act(async () => {
      rerender({ factoryConfigure: false });
    });
    await act(async () => {
      rerender({ factoryConfigure: true });
    });

    expect(activateCanvasVersionForEditing).toHaveBeenCalledTimes(2);
  });
});
