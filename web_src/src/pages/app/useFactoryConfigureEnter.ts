import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { useEffect, useRef, type Dispatch, type MutableRefObject, type SetStateAction } from "react";

type UseFactoryConfigureEnterOptions = {
  factoryConfigure: boolean;
  editSessionActive: boolean;
  setEditSessionActive: Dispatch<SetStateAction<boolean>>;
  canStageCanvasVersion: boolean;
  canvasLoading: boolean;
  liveCanvasVersionLoading: boolean;
  liveCanvasVersionId?: string;
  liveCanvasVersion?: CanvasesCanvasVersion | null;
  liveCanvas?: CanvasesCanvas | null;
  previewingCurrentVersionRef: MutableRefObject<boolean>;
  activateCanvasVersionForEditing: (
    versionId: string,
    version: CanvasesCanvasVersion,
    options?: { preserveStagedLayer?: boolean },
  ) => void;
  draftCanvasSpecsRef: MutableRefObject<Map<string, CanvasesCanvas["spec"] | null>>;
  setDraftCanvasSpec: Dispatch<SetStateAction<CanvasesCanvas["spec"] | null>>;
  resyncStagedEditorState: (
    versionId: string,
    options?: { bumpResetNonce?: boolean; preferCachedStagedSpec?: boolean },
  ) => Promise<void>;
  setLastSavedWorkflowSnapshot: (workflow: CanvasesCanvas | null) => void;
};

/** Seeds Configure edit session from live canvas, then awaits staged resync. */
export function useFactoryConfigureEnter({
  factoryConfigure,
  editSessionActive,
  setEditSessionActive,
  canStageCanvasVersion,
  canvasLoading,
  liveCanvasVersionLoading,
  liveCanvasVersionId,
  liveCanvasVersion,
  liveCanvas,
  previewingCurrentVersionRef,
  activateCanvasVersionForEditing,
  draftCanvasSpecsRef,
  setDraftCanvasSpec,
  resyncStagedEditorState,
  setLastSavedWorkflowSnapshot,
}: UseFactoryConfigureEnterOptions) {
  const factoryConfigureEnterStartedRef = useRef(false);

  useEffect(() => {
    if (!factoryConfigure) {
      factoryConfigureEnterStartedRef.current = false;
      return;
    }
    if (factoryConfigureEnterStartedRef.current || editSessionActive) {
      return;
    }
    if (!canStageCanvasVersion || !liveCanvasVersionId || !liveCanvasVersion) {
      return;
    }
    if (canvasLoading || liveCanvasVersionLoading) {
      return;
    }

    factoryConfigureEnterStartedRef.current = true;
    // Seed draft from the live version directly — do not wait on the versions
    // list (handleUseVersion), which often left Configure with no activeVersionId.
    const immediateSpec = liveCanvas?.spec ?? liveCanvasVersion.spec ?? { nodes: [], edges: [] };
    const versionForEdit: CanvasesCanvasVersion = {
      ...liveCanvasVersion,
      metadata: {
        ...liveCanvasVersion.metadata,
        id: liveCanvasVersionId,
      },
      spec: immediateSpec,
    };
    const configureVersionId = liveCanvasVersionId;
    let cancelled = false;
    let editEnabled = false;

    previewingCurrentVersionRef.current = true;
    activateCanvasVersionForEditing(configureVersionId, versionForEdit, { preserveStagedLayer: true });
    draftCanvasSpecsRef.current.set(configureVersionId, immediateSpec);
    setDraftCanvasSpec(immediateSpec);

    // Await staged resync before enabling edit so a late applyStagedSpec cannot
    // wipe edits typed against the immediate seed.
    void (async () => {
      try {
        await resyncStagedEditorState(configureVersionId, { bumpResetNonce: false });
      } catch {
        // Keep the immediate live/committed spec so Configure stays usable.
      }
      if (cancelled) {
        return;
      }
      editEnabled = true;
      setEditSessionActive(true);
      if (liveCanvas) {
        const spec = draftCanvasSpecsRef.current.get(configureVersionId) ?? immediateSpec;
        setLastSavedWorkflowSnapshot({ ...liveCanvas, spec });
      }
    })();

    return () => {
      cancelled = true;
      if (!editEnabled) {
        factoryConfigureEnterStartedRef.current = false;
      }
    };
  }, [
    activateCanvasVersionForEditing,
    canStageCanvasVersion,
    canvasLoading,
    draftCanvasSpecsRef,
    editSessionActive,
    factoryConfigure,
    liveCanvas,
    liveCanvasVersion,
    liveCanvasVersionId,
    liveCanvasVersionLoading,
    previewingCurrentVersionRef,
    resyncStagedEditorState,
    setDraftCanvasSpec,
    setEditSessionActive,
    setLastSavedWorkflowSnapshot,
  ]);
}
