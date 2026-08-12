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
  // Latest values via refs so the enter effect deps stay stable. Unstable
  // callback/query identities must not re-fire enter (update-depth loop), and
  // resync setQueryData must not cancel the in-flight path before edit enables.
  const liveCanvasRef = useRef(liveCanvas);
  liveCanvasRef.current = liveCanvas;
  const liveCanvasVersionRef = useRef(liveCanvasVersion);
  liveCanvasVersionRef.current = liveCanvasVersion;
  const enterDepsRef = useRef({
    activateCanvasVersionForEditing,
    draftCanvasSpecsRef,
    previewingCurrentVersionRef,
    resyncStagedEditorState,
    setDraftCanvasSpec,
    setEditSessionActive,
    setLastSavedWorkflowSnapshot,
  });
  enterDepsRef.current = {
    activateCanvasVersionForEditing,
    draftCanvasSpecsRef,
    previewingCurrentVersionRef,
    resyncStagedEditorState,
    setDraftCanvasSpec,
    setEditSessionActive,
    setLastSavedWorkflowSnapshot,
  };

  // Per Configure visit: bump when leaving so a later visit can seed again.
  // Once edit enables for a visit, save/exit must not re-seed while URL still
  // has configure=1 (handleCommittedVersionId clears editSessionActive first).
  const configureVisitIdRef = useRef(0);
  const inFlightVisitIdRef = useRef<number | null>(null);
  const editEnabledVisitIdRef = useRef<number | null>(null);

  useEffect(() => {
    if (factoryConfigure) {
      return;
    }
    configureVisitIdRef.current += 1;
    inFlightVisitIdRef.current = null;
    editEnabledVisitIdRef.current = null;
  }, [factoryConfigure]);

  useEffect(() => {
    if (!factoryConfigure || editSessionActive) {
      return;
    }
    if (!canStageCanvasVersion || !liveCanvasVersionId) {
      return;
    }
    if (canvasLoading || liveCanvasVersionLoading) {
      return;
    }

    const visitId = configureVisitIdRef.current;
    // Save/discard can clear editSessionActive while configure=1 is still set.
    // Never re-seed that same Configure visit.
    if (editEnabledVisitIdRef.current === visitId || inFlightVisitIdRef.current === visitId) {
      return;
    }

    const version = liveCanvasVersionRef.current;
    if (!version) {
      return;
    }

    inFlightVisitIdRef.current = visitId;
    // Seed draft from the live version directly — do not wait on the versions
    // list (handleUseVersion), which often left Configure with no activeVersionId.
    const immediateSpec = liveCanvasRef.current?.spec ?? version.spec ?? { nodes: [], edges: [] };
    const versionForEdit: CanvasesCanvasVersion = {
      ...version,
      metadata: {
        ...version.metadata,
        id: liveCanvasVersionId,
      },
      spec: immediateSpec,
    };
    const configureVersionId = liveCanvasVersionId;
    let cancelled = false;
    const {
      activateCanvasVersionForEditing: activate,
      draftCanvasSpecsRef: specsRef,
      previewingCurrentVersionRef: previewingRef,
      resyncStagedEditorState: resync,
      setDraftCanvasSpec: setDraft,
      setEditSessionActive: setEditActive,
      setLastSavedWorkflowSnapshot: setSnapshot,
    } = enterDepsRef.current;

    previewingRef.current = true;
    activate(configureVersionId, versionForEdit, { preserveStagedLayer: true });
    specsRef.current.set(configureVersionId, immediateSpec);
    setDraft(immediateSpec);

    // Await staged resync before enabling edit so a late applyStagedSpec cannot
    // wipe edits typed against the immediate seed.
    void (async () => {
      try {
        await resync(configureVersionId, { bumpResetNonce: false });
      } catch {
        // Keep the immediate live/committed spec so Configure stays usable.
      }
      if (cancelled || configureVisitIdRef.current !== visitId) {
        return;
      }
      editEnabledVisitIdRef.current = visitId;
      inFlightVisitIdRef.current = null;
      setEditActive(true);
      const canvas = liveCanvasRef.current;
      if (canvas) {
        const spec = specsRef.current.get(configureVersionId) ?? immediateSpec;
        setSnapshot({ ...canvas, spec });
      }
    })();

    return () => {
      cancelled = true;
      // Strict Mode remount: allow retry only when edit never enabled for visit.
      if (editEnabledVisitIdRef.current !== visitId && inFlightVisitIdRef.current === visitId) {
        inFlightVisitIdRef.current = null;
      }
    };
  }, [
    canStageCanvasVersion,
    canvasLoading,
    editSessionActive,
    factoryConfigure,
    liveCanvasVersionId,
    liveCanvasVersionLoading,
  ]);
}
