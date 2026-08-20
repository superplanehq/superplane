import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type Dispatch,
  type MutableRefObject,
  type SetStateAction,
} from "react";

import { startFactoryConfigureEnter, type FactoryConfigureEnterDeps } from "./factoryConfigureEnterSession";

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
  activateCanvasVersionForEditing: FactoryConfigureEnterDeps["activateCanvasVersionForEditing"];
  draftCanvasSpecsRef: MutableRefObject<Map<string, CanvasesCanvas["spec"] | null>>;
  setDraftCanvasSpec: Dispatch<SetStateAction<CanvasesCanvas["spec"] | null>>;
  resyncStagedEditorState: FactoryConfigureEnterDeps["resyncStagedEditorState"];
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
  const enterDepsRef = useRef<FactoryConfigureEnterDeps>({
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
  // Once edit enables for a visit, clearing editSessionActive with configure=1
  // still set must not re-seed that visit (save teardown / discard). Save-and-stay
  // calls allowNextConfigureEnter so the next seed uses a new visit.
  const configureVisitIdRef = useRef(0);
  const inFlightVisitIdRef = useRef<number | null>(null);
  const editEnabledVisitIdRef = useRef<number | null>(null);
  const [configureEnterNonce, setConfigureEnterNonce] = useState(0);

  const allowNextConfigureEnter = useCallback(() => {
    configureVisitIdRef.current += 1;
    inFlightVisitIdRef.current = null;
    editEnabledVisitIdRef.current = null;
    setConfigureEnterNonce((nonce) => nonce + 1);
  }, []);

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
    return startFactoryConfigureEnter({
      visitId,
      configureVersionId: liveCanvasVersionId,
      version,
      liveCanvasRef,
      configureVisitIdRef,
      inFlightVisitIdRef,
      editEnabledVisitIdRef,
      deps: enterDepsRef.current,
    });
  }, [
    canStageCanvasVersion,
    canvasLoading,
    editSessionActive,
    factoryConfigure,
    liveCanvasVersionId,
    liveCanvasVersionLoading,
    configureEnterNonce,
  ]);

  return { allowNextConfigureEnter };
}
