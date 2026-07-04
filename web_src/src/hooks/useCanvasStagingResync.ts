import { useCallback, useRef, type Dispatch, type MutableRefObject, type SetStateAction } from "react";
import { useQueryClient } from "@tanstack/react-query";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { fetchCanvasVersionWithSpec } from "@/pages/app/lib/repository-spec-files";

import { canvasKeys } from "@/hooks/useCanvasData";

type CanvasSpec = CanvasesCanvas["spec"] | null;

interface UseCanvasStagingResyncOptions {
  organizationId?: string;
  canvasId?: string;
  activeCanvasVersionIdRef: MutableRefObject<string>;
  draftCanvasSpecsRef: MutableRefObject<Map<string, CanvasSpec>>;
  consoleMutationGenerationRef: MutableRefObject<number>;
  setDraftCanvasSpec: Dispatch<SetStateAction<CanvasSpec>>;
  setActiveCanvasVersion: Dispatch<SetStateAction<CanvasesCanvasVersion | null>>;
  setLastSavedWorkflowSnapshot: (workflow: CanvasesCanvas | null) => void;
  setStagingResetNonce: Dispatch<SetStateAction<number>>;
}

type ResyncStagedOptions = {
  /** Bumps stagingResetNonce so file/console baselines reset. Avoid when entering edit from agent staging — it remounts CanvasPage and can loop auto-open. */
  bumpResetNonce?: boolean;
};

// Re-applies the staged (uncommitted) spec into React editor state after a remote
// staging_updated event. The graph reads local state, not React Query, so query
// invalidation alone is not enough while the user is actively editing.
export function useCanvasStagingResync(options: UseCanvasStagingResyncOptions) {
  const {
    organizationId,
    canvasId,
    activeCanvasVersionIdRef,
    draftCanvasSpecsRef,
    consoleMutationGenerationRef,
    setDraftCanvasSpec,
    setActiveCanvasVersion,
    setLastSavedWorkflowSnapshot,
    setStagingResetNonce,
  } = options;
  const queryClient = useQueryClient();
  const resyncStagedInFlightRef = useRef(new Map<string, Promise<void>>());

  const applyStagedSpec = useCallback(
    (versionId: string, spec: CanvasSpec) => {
      if (!organizationId || !canvasId) {
        return;
      }

      if (spec) {
        draftCanvasSpecsRef.current.set(versionId, spec);
      } else {
        draftCanvasSpecsRef.current.delete(versionId);
      }
      setDraftCanvasSpec(spec);
      setActiveCanvasVersion((current) =>
        current?.metadata?.id === versionId ? { ...current, spec: spec ?? current.spec } : current,
      );

      const restored = queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId));
      if (restored) {
        setLastSavedWorkflowSnapshot(restored);
      }
    },
    [
      organizationId,
      canvasId,
      queryClient,
      draftCanvasSpecsRef,
      setDraftCanvasSpec,
      setActiveCanvasVersion,
      setLastSavedWorkflowSnapshot,
    ],
  );

  const resyncStagedEditorState = useCallback(
    async (versionId: string, options?: ResyncStagedOptions) => {
      if (!organizationId || !canvasId) {
        return;
      }

      const inFlight = resyncStagedInFlightRef.current.get(versionId);
      if (inFlight) {
        await inFlight;
        return;
      }

      const bumpResetNonce = options?.bumpResetNonce ?? true;

      const resyncPromise = (async () => {
        await queryClient.invalidateQueries({ queryKey: canvasKeys.canvasStaging(canvasId) });

        if (activeCanvasVersionIdRef.current !== versionId) {
          draftCanvasSpecsRef.current.delete(versionId);
          return;
        }

        consoleMutationGenerationRef.current += 1;
        const stagedVersion = await fetchCanvasVersionWithSpec(canvasId, versionId, true);
        const stagedSpec = stagedVersion?.spec ?? null;

        if (stagedVersion) {
          queryClient.setQueryData(canvasKeys.versionStagedDetail(canvasId, versionId), stagedVersion);
        }
        if (stagedSpec) {
          queryClient.setQueryData<CanvasesCanvas | undefined>(
            canvasKeys.detail(organizationId, canvasId),
            (current) => (current ? { ...current, spec: { ...current.spec, ...stagedSpec } } : current),
          );
        }
        applyStagedSpec(versionId, stagedSpec);

        await queryClient.invalidateQueries({ queryKey: canvasKeys.consoleStaged(canvasId, versionId) });
        if (bumpResetNonce) {
          setStagingResetNonce((nonce) => nonce + 1);
        }
      })();

      resyncStagedInFlightRef.current.set(versionId, resyncPromise);
      try {
        await resyncPromise;
      } finally {
        resyncStagedInFlightRef.current.delete(versionId);
      }
    },
    [
      organizationId,
      canvasId,
      queryClient,
      activeCanvasVersionIdRef,
      draftCanvasSpecsRef,
      consoleMutationGenerationRef,
      applyStagedSpec,
      setStagingResetNonce,
    ],
  );

  return { resyncStagedEditorState };
}

/** @deprecated Use useCanvasStagingResync */
export const useCanvasDraftResync = useCanvasStagingResync;
