import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { useFactoryConfigureEnter } from "./useFactoryConfigureEnter";

function baseOptions(overrides: Partial<Parameters<typeof useFactoryConfigureEnter>[0]> = {}) {
  return {
    factoryConfigure: true,
    editSessionActive: false,
    setEditSessionActive: vi.fn(),
    canStageCanvasVersion: true,
    canvasLoading: false,
    liveCanvasVersionLoading: false,
    liveCanvasVersionId: "version-live",
    liveCanvasVersion: { metadata: { id: "version-live" }, spec: { nodes: [], edges: [] } },
    liveCanvas: { metadata: { id: "canvas-1", liveVersionId: "version-live" }, spec: { nodes: [], edges: [] } },
    previewingCurrentVersionRef: { current: false },
    activateCanvasVersionForEditing: vi.fn(),
    draftCanvasSpecsRef: { current: new Map() },
    setDraftCanvasSpec: vi.fn(),
    resyncStagedEditorState: vi.fn().mockResolvedValue(undefined),
    setLastSavedWorkflowSnapshot: vi.fn(),
    ...overrides,
  };
}

describe("useFactoryConfigureEnter", () => {
  it("does not re-enter Configure when activateCanvasVersionForEditing identity changes before edit is enabled", async () => {
    const activateCanvasVersionForEditing = vi.fn();
    const resyncStagedEditorState = vi.fn(() => new Promise<void>(() => {}));

    const { rerender } = renderHook(
      ({ activate }: { activate: typeof activateCanvasVersionForEditing }) =>
        useFactoryConfigureEnter(
          baseOptions({
            activateCanvasVersionForEditing: activate,
            resyncStagedEditorState,
          }),
        ),
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

  it("does not cancel Configure enter when liveCanvas identity changes mid-resync", async () => {
    const setEditSessionActive = vi.fn();
    let resolveResync!: () => void;
    const resyncStagedEditorState = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveResync = resolve;
        }),
    );

    const { rerender } = renderHook(
      ({ liveCanvas }: { liveCanvas: NonNullable<Parameters<typeof useFactoryConfigureEnter>[0]["liveCanvas"]> }) =>
        useFactoryConfigureEnter(
          baseOptions({
            liveCanvas,
            setEditSessionActive,
            resyncStagedEditorState,
          }),
        ),
      {
        initialProps: {
          liveCanvas: {
            metadata: { id: "canvas-1", liveVersionId: "version-live" },
            spec: { nodes: [], edges: [] },
          },
        },
      },
    );

    expect(resyncStagedEditorState).toHaveBeenCalledTimes(1);

    await act(async () => {
      // New liveCanvas object identity (as after setQueryData), same shape.
      rerender({
        liveCanvas: {
          metadata: { id: "canvas-1", liveVersionId: "version-live" },
          spec: { nodes: [], edges: [] },
        },
      });
    });

    await act(async () => {
      resolveResync();
    });

    await waitFor(() => {
      expect(setEditSessionActive).toHaveBeenCalledWith(true);
    });
    expect(resyncStagedEditorState).toHaveBeenCalledTimes(1);
  });

  it("allows a fresh Configure enter after leaving configure mode", async () => {
    const activateCanvasVersionForEditing = vi.fn();

    const { rerender } = renderHook(
      ({ factoryConfigure }: { factoryConfigure: boolean }) =>
        useFactoryConfigureEnter(
          baseOptions({
            factoryConfigure,
            activateCanvasVersionForEditing,
          }),
        ),
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

  it("does not re-enter Configure when editSessionActive clears during save exit", async () => {
    const activateCanvasVersionForEditing = vi.fn();
    const setEditSessionActive = vi.fn();

    const { rerender } = renderHook(
      ({ editSessionActive, factoryConfigure }: { editSessionActive: boolean; factoryConfigure: boolean }) =>
        useFactoryConfigureEnter(
          baseOptions({
            editSessionActive,
            factoryConfigure,
            activateCanvasVersionForEditing,
            setEditSessionActive,
          }),
        ),
      { initialProps: { editSessionActive: false, factoryConfigure: true } },
    );

    await waitFor(() => {
      expect(setEditSessionActive).toHaveBeenCalledWith(true);
    });
    expect(activateCanvasVersionForEditing).toHaveBeenCalledTimes(1);

    await act(async () => {
      rerender({ editSessionActive: true, factoryConfigure: true });
    });
    // Save path: flushSync clears edit while configure=1 still present.
    await act(async () => {
      rerender({ editSessionActive: false, factoryConfigure: true });
    });

    expect(activateCanvasVersionForEditing).toHaveBeenCalledTimes(1);
    expect(setEditSessionActive).toHaveBeenCalledTimes(1);
  });
});
