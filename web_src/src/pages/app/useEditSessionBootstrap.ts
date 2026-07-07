import { useRef } from "react";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";

import {
  isDraftCanvasLoadingWhileEditing,
  isEditBootstrapReady as resolveEditBootstrapReady,
} from "./lib/edit-staging-ready";
import { useCommittedDraftBaselines } from "./useCommittedDraftBaselines";

type UseEditSessionBootstrapOptions = {
  canvasId: string;
  isEditing: boolean;
  isEnteringEditSession: boolean;
  shouldReadStagedCanvasVersion: boolean;
  activeCanvasVersionId: string;
  stagingResetNonce: number;
  draftCanvasSpec: CanvasesCanvas["spec"] | null;
  draftSpecToRender: CanvasesCanvas["spec"] | null;
  loadedCanvasVersionLoading: boolean;
  loadedCanvasVersionFetching: boolean;
  selectedCanvasVersion: CanvasesCanvasVersion | null;
  liveCanvasVersionId?: string;
  isRunInspectionMode: boolean;
};

export function useEditSessionBootstrap({
  canvasId,
  isEditing,
  isEnteringEditSession,
  shouldReadStagedCanvasVersion,
  activeCanvasVersionId,
  stagingResetNonce,
  draftCanvasSpec,
  draftSpecToRender,
  loadedCanvasVersionLoading,
  loadedCanvasVersionFetching,
  selectedCanvasVersion,
  liveCanvasVersionId,
  isRunInspectionMode,
}: UseEditSessionBootstrapOptions) {
  const stableCanvasViewKeyRef = useRef("live");
  const committedBaselinesForEdit = useCommittedDraftBaselines({
    canvasId,
    versionId: activeCanvasVersionId || undefined,
    enabled: isEditing,
    stagingResetNonce,
  });
  const isEditBootstrapReady = resolveEditBootstrapReady({
    isEditing,
    isEnteringEditSession,
    stagingBaselinesReady: committedBaselinesForEdit.ready,
    draftCanvasSpec,
    shouldReadStagedCanvasVersion,
  });
  const draftSpecForView = isEditing && !isEditBootstrapReady ? null : draftSpecToRender;
  const isDraftCanvasLoading = isDraftCanvasLoadingWhileEditing({
    isEditing,
    shouldReadStagedCanvasVersion,
    isEnteringEditSession,
    isEditBootstrapReady,
    draftCanvasSpec,
    loadedCanvasVersionLoading,
    loadedCanvasVersionFetching,
  });
  const isEditSessionUiReady = !isEditing || (isEditBootstrapReady && !isDraftCanvasLoading);
  const canvasViewKey = selectedCanvasVersion?.metadata?.id || liveCanvasVersionId || "live";
  const pinCanvasViewKey = isEnteringEditSession || (isEditing && !isEditBootstrapReady);
  if (!pinCanvasViewKey) {
    stableCanvasViewKeyRef.current = canvasViewKey;
  }
  const stableCanvasViewKey = pinCanvasViewKey ? stableCanvasViewKeyRef.current : canvasViewKey;
  const canvasRenderKey = `${stableCanvasViewKey}:${isRunInspectionMode ? "runs" : "canvas"}:reset-${stagingResetNonce}`;

  return {
    committedBaselinesForEdit,
    isEditBootstrapReady,
    draftSpecForView,
    isDraftCanvasLoading,
    isEditSessionUiReady,
    stableCanvasViewKey,
    canvasRenderKey,
  };
}
