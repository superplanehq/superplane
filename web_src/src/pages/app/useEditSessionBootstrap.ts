import { useRef } from "react";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";

import {
  isDraftCanvasLoadingWhileEditing,
  isEditBootstrapReady as resolveEditBootstrapReady,
} from "./lib/edit-staging-ready";
import { useCommittedDraftBaselines } from "./useCommittedDraftBaselines";

function resolveStableCanvasViewKey(
  stableCanvasViewKeyRef: { current: string },
  canvasViewKey: string,
  pinCanvasViewKey: boolean,
) {
  if (!pinCanvasViewKey) {
    stableCanvasViewKeyRef.current = canvasViewKey;
  }
  return pinCanvasViewKey ? stableCanvasViewKeyRef.current : canvasViewKey;
}

function buildCanvasRenderKey(stableCanvasViewKey: string, isRunInspectionMode: boolean, stagingResetNonce: number) {
  const mode = isRunInspectionMode ? "runs" : "canvas";
  return `${stableCanvasViewKey}:${mode}:reset-${stagingResetNonce}`;
}

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
  includeConsoleBaseline?: boolean;
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
  includeConsoleBaseline = true,
}: UseEditSessionBootstrapOptions) {
  const stableCanvasViewKeyRef = useRef("live");
  const committedBaselinesForEdit = useCommittedDraftBaselines({
    canvasId,
    versionId: activeCanvasVersionId || undefined,
    enabled: isEditing,
    stagingResetNonce,
    includeConsole: includeConsoleBaseline,
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
  const stableCanvasViewKey = resolveStableCanvasViewKey(stableCanvasViewKeyRef, canvasViewKey, pinCanvasViewKey);
  const canvasRenderKey = buildCanvasRenderKey(stableCanvasViewKey, isRunInspectionMode, stagingResetNonce);

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
