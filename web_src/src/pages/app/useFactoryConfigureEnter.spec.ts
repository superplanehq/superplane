import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { FACTORY_CONFIGURE_ENTER_RESYNC_TIMEOUT_MS } from "./factoryConfigureEnterSession";
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
  afterEach(() => {
    vi.useRealTimers();
  });
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

  it("re-enters Configure after save when the next visit is allowed", async () => {
    const activateCanvasVersionForEditing = vi.fn();
    const setEditSessionActive = vi.fn();

    const { rerender, result } = renderHook(
      ({ editSessionActive }: { editSessionActive: boolean }) =>
        useFactoryConfigureEnter(
          baseOptions({
            editSessionActive,
            factoryConfigure: true,
            activateCanvasVersionForEditing,
            setEditSessionActive,
          }),
        ),
      { initialProps: { editSessionActive: false } },
    );

    await waitFor(() => {
      expect(setEditSessionActive).toHaveBeenCalledWith(true);
    });
    expect(activateCanvasVersionForEditing).toHaveBeenCalledTimes(1);

    await act(async () => {
      rerender({ editSessionActive: true });
    });
    await act(async () => {
      result.current.allowNextConfigureEnter();
      rerender({ editSessionActive: false });
    });

    await waitFor(() => {
      expect(activateCanvasVersionForEditing).toHaveBeenCalledTimes(2);
    });
  });

  it("enables edit when staged resync does not finish in time", async () => {
    vi.useFakeTimers();
    const setEditSessionActive = vi.fn();
    const resyncStagedEditorState = vi.fn(() => new Promise<void>(() => {}));

    renderHook(() =>
      useFactoryConfigureEnter(
        baseOptions({
          setEditSessionActive,
          resyncStagedEditorState,
        }),
      ),
    );

    expect(setEditSessionActive).not.toHaveBeenCalled();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(FACTORY_CONFIGURE_ENTER_RESYNC_TIMEOUT_MS);
    });

    expect(setEditSessionActive).toHaveBeenCalledWith(true);
  });

  it("does not apply a late staged resync after the enter timeout", async () => {
    vi.useFakeTimers();
    const setEditSessionActive = vi.fn();
    const setDraftCanvasSpec = vi.fn();
    const lateSpec = { nodes: [{ id: "late" }], edges: [] };

    const resyncStagedEditorState = vi.fn((_versionId: string, options?: { signal?: AbortSignal }) => {
      return new Promise<void>((resolve) => {
        const finish = () => {
          if (!options?.signal?.aborted) {
            setDraftCanvasSpec(lateSpec);
          }
          resolve();
        };
        const timeoutId = window.setTimeout(finish, 5000);
        options?.signal?.addEventListener("abort", () => {
          window.clearTimeout(timeoutId);
          finish();
        });
      });
    });

    renderHook(() =>
      useFactoryConfigureEnter(
        baseOptions({
          setEditSessionActive,
          setDraftCanvasSpec,
          resyncStagedEditorState,
        }),
      ),
    );

    expect(setDraftCanvasSpec).toHaveBeenCalled();
    const seedCallCount = setDraftCanvasSpec.mock.calls.length;
    expect(resyncStagedEditorState).toHaveBeenCalledWith(
      "version-live",
      expect.objectContaining({ bumpResetNonce: false, signal: expect.any(AbortSignal) }),
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(FACTORY_CONFIGURE_ENTER_RESYNC_TIMEOUT_MS);
    });

    expect(setEditSessionActive).toHaveBeenCalledWith(true);
    const lastCall = resyncStagedEditorState.mock.calls.at(-1);
    const signal = lastCall?.[1]?.signal;
    expect(signal?.aborted).toBe(true);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });

    expect(setDraftCanvasSpec).toHaveBeenCalledTimes(seedCallCount);
    expect(setDraftCanvasSpec.mock.calls.some((call) => call[0] === lateSpec)).toBe(false);
  });
});
