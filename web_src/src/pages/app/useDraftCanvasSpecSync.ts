import type { Dispatch, MutableRefObject, SetStateAction } from "react";
import { useEffect } from "react";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";

import {
  shouldApplyPreservedDraftSpec,
  shouldPreserveDraftSpec,
  shouldSkipDraftSpecSyncFromLoadedVersion,
} from "./lib/draft-canvas-sync";

type DraftSpec = CanvasesCanvas["spec"] | null;

type UseDraftCanvasSpecSyncOptions = {
  isEditing: boolean;
  isEnteringEditSession: boolean;
  shouldReadStagedCanvasVersion: boolean;
  awaitingCanvasStaging: boolean;
  activeCanvasVersionId: string;
  selectedCanvasVersion: CanvasesCanvasVersion | null;
  draftCanvasSpec: DraftSpec;
  setDraftCanvasSpec: Dispatch<SetStateAction<DraftSpec>>;
  draftCanvasSpecsRef: MutableRefObject<Map<string, DraftSpec>>;
  liveCanvas?: CanvasesCanvas | null;
  liveCanvasVersion?: CanvasesCanvasVersion;
};

export function useDraftCanvasSpecSync({
  isEditing,
  isEnteringEditSession,
  shouldReadStagedCanvasVersion,
  awaitingCanvasStaging,
  activeCanvasVersionId,
  selectedCanvasVersion,
  draftCanvasSpec,
  setDraftCanvasSpec,
  draftCanvasSpecsRef,
  liveCanvas,
  liveCanvasVersion,
}: UseDraftCanvasSpecSyncOptions) {
  useEffect(() => {
    if (!isEditing || !activeCanvasVersionId) {
      return;
    }

    // Live staged edit draft state is owned by resyncStagedEditorState (remote
    // staging) and local canvas mutations — not by React Query sync effects.
    if (shouldReadStagedCanvasVersion) {
      return;
    }

    if (isEnteringEditSession || awaitingCanvasStaging) {
      return;
    }

    const nextDraftSpec = selectedCanvasVersion?.spec ?? null;
    if (!nextDraftSpec) {
      return;
    }

    if (shouldSkipDraftSpecSyncFromLoadedVersion(draftCanvasSpec, nextDraftSpec)) {
      return;
    }

    const preservedDraftSpec = draftCanvasSpecsRef.current.get(activeCanvasVersionId);

    if (shouldApplyPreservedDraftSpec(preservedDraftSpec, nextDraftSpec)) {
      setDraftCanvasSpec(preservedDraftSpec);
      return;
    }

    draftCanvasSpecsRef.current.set(activeCanvasVersionId, nextDraftSpec);
    setDraftCanvasSpec(nextDraftSpec);
  }, [
    isEditing,
    isEnteringEditSession,
    shouldReadStagedCanvasVersion,
    awaitingCanvasStaging,
    activeCanvasVersionId,
    selectedCanvasVersion?.metadata?.id,
    selectedCanvasVersion?.spec,
    draftCanvasSpec,
    setDraftCanvasSpec,
    draftCanvasSpecsRef,
  ]);

  useEffect(() => {
    if (!isEditing || !activeCanvasVersionId || !liveCanvas?.spec) {
      return;
    }

    if (shouldReadStagedCanvasVersion) {
      return;
    }

    if (!draftCanvasSpec && !selectedCanvasVersion?.spec) {
      return;
    }

    if (
      shouldPreserveDraftSpec({
        incomingSpec: liveCanvas.spec,
        draftSpec: draftCanvasSpec,
        selectedDraftVersionSpec: selectedCanvasVersion?.spec,
        liveVersionSpec: liveCanvasVersion?.spec,
      })
    ) {
      return;
    }

    setDraftCanvasSpec((currentDraftSpec) => {
      draftCanvasSpecsRef.current.set(activeCanvasVersionId, liveCanvas.spec);
      if (currentDraftSpec === liveCanvas.spec) {
        return currentDraftSpec;
      }

      return liveCanvas.spec;
    });
  }, [
    isEditing,
    shouldReadStagedCanvasVersion,
    activeCanvasVersionId,
    liveCanvas?.spec,
    liveCanvasVersion?.spec,
    selectedCanvasVersion?.spec,
    draftCanvasSpec,
    setDraftCanvasSpec,
    draftCanvasSpecsRef,
  ]);
}
