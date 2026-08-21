import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import type { QueryClient } from "@tanstack/react-query";
import { useCallback, useRef, type Dispatch, type MutableRefObject, type SetStateAction } from "react";
import type { SetURLSearchParams } from "react-router";

import {
  activateCanvasVersionForEditing as applyCanvasVersionForEditing,
  type ActivateCanvasVersionOptions,
} from "./lib/canvas-version-activation";

type UseActivateCanvasVersionForEditingOptions = {
  organizationId?: string;
  canvasId?: string;
  liveCanvasVersionId?: string;
  queryClient: QueryClient;
  draftCanvasSpec: CanvasesCanvas["spec"] | null;
  draftCanvasSpecsRef: MutableRefObject<Map<string, CanvasesCanvas["spec"] | null>>;
  activeCanvasVersionIdRef: MutableRefObject<string>;
  lastAppliedVersionSnapshotRef: MutableRefObject<string>;
  liveCanvasVersion?: CanvasesCanvasVersion | null;
  liveCanvas?: CanvasesCanvas | null;
  clearPendingAutoSaveWork: () => void;
  setDraftCanvasSpec: Dispatch<SetStateAction<CanvasesCanvas["spec"] | null>>;
  setActiveCanvasVersion: Dispatch<SetStateAction<CanvasesCanvasVersion | null>>;
  setLastSavedWorkflowSnapshot: (workflow: CanvasesCanvas | null) => void;
  setSearchParams: SetURLSearchParams;
  initializeFromWorkflow: (canvas: CanvasesCanvas) => void;
};

/**
 * Stable activate helper — reads draftCanvasSpec via ref so Configure seeding
 * does not recreate the callback and re-trigger enter effects.
 */
export function useActivateCanvasVersionForEditing({
  organizationId,
  canvasId,
  liveCanvasVersionId,
  queryClient,
  draftCanvasSpec,
  draftCanvasSpecsRef,
  activeCanvasVersionIdRef,
  lastAppliedVersionSnapshotRef,
  liveCanvasVersion,
  liveCanvas,
  clearPendingAutoSaveWork,
  setDraftCanvasSpec,
  setActiveCanvasVersion,
  setLastSavedWorkflowSnapshot,
  setSearchParams,
  initializeFromWorkflow,
}: UseActivateCanvasVersionForEditingOptions) {
  const draftCanvasSpecRef = useRef(draftCanvasSpec);
  draftCanvasSpecRef.current = draftCanvasSpec;

  return useCallback(
    (versionID: string, version: CanvasesCanvasVersion, options?: ActivateCanvasVersionOptions) =>
      applyCanvasVersionForEditing({
        organizationId,
        canvasId,
        versionID,
        version,
        options,
        liveCanvasVersionId,
        queryClient,
        draftCanvasSpec: draftCanvasSpecRef.current,
        draftCanvasSpecsRef,
        activeCanvasVersionIdRef,
        lastAppliedVersionSnapshotRef,
        liveCanvasVersion: liveCanvasVersion ?? undefined,
        liveCanvas,
        clearPendingAutoSaveWork,
        setDraftCanvasSpec,
        setActiveCanvasVersion,
        setLastSavedWorkflowSnapshot,
        setSearchParams,
        initializeFromWorkflow,
      }),
    [
      organizationId,
      canvasId,
      liveCanvasVersionId,
      queryClient,
      draftCanvasSpecsRef,
      activeCanvasVersionIdRef,
      lastAppliedVersionSnapshotRef,
      liveCanvasVersion,
      liveCanvas,
      clearPendingAutoSaveWork,
      setDraftCanvasSpec,
      setActiveCanvasVersion,
      setLastSavedWorkflowSnapshot,
      setSearchParams,
      initializeFromWorkflow,
    ],
  );
}
