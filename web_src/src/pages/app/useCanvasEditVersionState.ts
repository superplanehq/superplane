import { useMemo } from "react";

import type { CanvasesCanvasVersion } from "@/api-client";
import { useCanvasVersion, useStagedCanvasSpec } from "@/hooks/useCanvasData";

import { canvasVersionShell, type CanvasVersionListItem } from "./lib/canvas-versions";
import {
  isAwaitingStagedCanvasSpec,
  isViewingCurrentLiveCanvasVersion,
  resolveSelectedCanvasVersion,
  shouldReadStagedCanvasVersion,
} from "./lib/live-edit-session";

type UseCanvasEditVersionStateOptions = {
  organizationId: string;
  canvasId: string;
  editSessionActive: boolean;
  isEnteringEditSession: boolean;
  activeCanvasVersion: CanvasesCanvasVersion | null;
  liveCanvasVersionId?: string;
  selectableVersionsById: Map<string, CanvasVersionListItem>;
  isRunInspectionMode: boolean;
  isMemoryMode: boolean;
};

export function useCanvasEditVersionState({
  organizationId,
  canvasId,
  editSessionActive,
  isEnteringEditSession,
  activeCanvasVersion,
  liveCanvasVersionId,
  selectableVersionsById,
  isRunInspectionMode,
  isMemoryMode,
}: UseCanvasEditVersionStateOptions) {
  const activeCanvasVersionId = activeCanvasVersion?.metadata?.id || "";
  const shouldReadStagedCanvasVersionFlag = shouldReadStagedCanvasVersion({
    editSessionActive,
    activeCanvasVersionId,
    liveCanvasVersionId,
  });
  const stagedVersionMetadataShell =
    activeCanvasVersion ?? canvasVersionShell(selectableVersionsById.get(activeCanvasVersionId)) ?? null;
  const {
    data: loadedStagedCanvasVersion,
    isLoading: loadedStagedCanvasVersionLoading,
    isFetching: loadedStagedCanvasVersionFetching,
  } = useStagedCanvasSpec(canvasId, stagedVersionMetadataShell, shouldReadStagedCanvasVersionFlag);
  const {
    data: loadedCommittedCanvasVersion,
    isLoading: loadedCommittedCanvasVersionLoading,
    isFetching: loadedCommittedCanvasVersionFetching,
  } = useCanvasVersion(
    organizationId,
    canvasId,
    activeCanvasVersionId,
    !!activeCanvasVersionId && !shouldReadStagedCanvasVersionFlag,
  );
  const loadedCanvasVersion = shouldReadStagedCanvasVersionFlag
    ? loadedStagedCanvasVersion
    : loadedCommittedCanvasVersion;
  const loadedCanvasVersionLoading = shouldReadStagedCanvasVersionFlag
    ? loadedStagedCanvasVersionLoading
    : loadedCommittedCanvasVersionLoading;
  const loadedCanvasVersionFetching = shouldReadStagedCanvasVersionFlag
    ? loadedStagedCanvasVersionFetching
    : loadedCommittedCanvasVersionFetching;
  const isAwaitingStagedCanvasSpecFlag = isAwaitingStagedCanvasSpec({
    activeCanvasVersionId,
    shouldReadStagedCanvasVersion: shouldReadStagedCanvasVersionFlag,
    loadedStagedCanvasVersion,
    loadedStagedCanvasVersionLoading,
    loadedStagedCanvasVersionFetching,
    isEnteringEditSession,
  });
  const selectedCanvasVersion = useMemo(
    () =>
      resolveSelectedCanvasVersion({
        activeCanvasVersionId,
        shouldReadStagedCanvasVersion: shouldReadStagedCanvasVersionFlag,
        loadedStagedCanvasVersion,
        loadedCommittedCanvasVersion,
        activeCanvasVersion,
        isAwaitingStagedSpec: isAwaitingStagedCanvasSpecFlag,
      }),
    [
      activeCanvasVersionId,
      shouldReadStagedCanvasVersionFlag,
      loadedStagedCanvasVersion,
      loadedCommittedCanvasVersion,
      activeCanvasVersion,
      isAwaitingStagedCanvasSpecFlag,
    ],
  );
  const isViewingCurrentLiveVersion = isViewingCurrentLiveCanvasVersion({
    activeCanvasVersionId,
    selectedCanvasVersion,
    liveCanvasVersionId,
  });
  const isEditing = editSessionActive && isViewingCurrentLiveVersion;
  const showLiveActivity = isViewingCurrentLiveVersion && !(editSessionActive && !isRunInspectionMode && !isMemoryMode);

  return {
    activeCanvasVersionId,
    shouldReadStagedCanvasVersionFlag,
    loadedCanvasVersion,
    loadedCanvasVersionLoading,
    loadedCanvasVersionFetching,
    isAwaitingStagedCanvasSpecFlag,
    selectedCanvasVersion,
    isViewingCurrentLiveVersion,
    isViewingLiveVersion: isViewingCurrentLiveVersion,
    isEditing,
    hasEditableVersion: isEditing,
    showLiveActivity,
  };
}
