import { useMemo } from "react";

import type { CanvasesCanvasVersion } from "@/api-client";
import { useCanvasStaging, useCanvasVersion } from "@/hooks/useCanvasData";

import {
  isAwaitingCanvasStaging,
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
  effectiveLiveCanvasVersionId?: string;
  liveCanvasVersionId?: string;
  selectableVersionsById: Map<string, CanvasesCanvasVersion>;
  isRunInspectionMode: boolean;
  isMemoryMode: boolean;
};

export function useCanvasEditVersionState({
  organizationId,
  canvasId,
  editSessionActive,
  isEnteringEditSession,
  activeCanvasVersion,
  effectiveLiveCanvasVersionId,
  liveCanvasVersionId,
  isRunInspectionMode,
  isMemoryMode,
}: UseCanvasEditVersionStateOptions) {
  const activeCanvasVersionId = activeCanvasVersion?.metadata?.id || "";
  const shouldReadStagedCanvasVersionFlag = shouldReadStagedCanvasVersion({
    editSessionActive,
    activeCanvasVersionId,
    effectiveLiveCanvasVersionId,
    liveCanvasVersionId,
  });
  const shouldLoadStaging = shouldReadStagedCanvasVersionFlag || isEnteringEditSession;
  const canvasStagingQuery = useCanvasStaging(canvasId, shouldLoadStaging);
  const { data: staging, isLoading: stagingLoading, isFetching: stagingFetching } = canvasStagingQuery;
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
  const loadedCanvasVersion = shouldReadStagedCanvasVersionFlag ? activeCanvasVersion : loadedCommittedCanvasVersion;
  const loadedCanvasVersionLoading = shouldReadStagedCanvasVersionFlag
    ? stagingLoading
    : loadedCommittedCanvasVersionLoading;
  const loadedCanvasVersionFetching = shouldReadStagedCanvasVersionFlag
    ? stagingFetching
    : loadedCommittedCanvasVersionFetching;
  const awaitingCanvasStaging = isAwaitingCanvasStaging({
    shouldReadStagedCanvasVersion: shouldReadStagedCanvasVersionFlag,
    stagingLoading,
    stagingFetching,
    isEnteringEditSession,
    staging,
  });
  const selectedCanvasVersion = useMemo(
    () =>
      resolveSelectedCanvasVersion({
        activeCanvasVersionId,
        shouldReadStagedCanvasVersion: shouldReadStagedCanvasVersionFlag,
        staging,
        loadedCommittedCanvasVersion,
        activeCanvasVersion,
        awaitingCanvasStaging,
      }),
    [
      activeCanvasVersionId,
      shouldReadStagedCanvasVersionFlag,
      staging,
      loadedCommittedCanvasVersion,
      activeCanvasVersion,
      awaitingCanvasStaging,
    ],
  );
  const isViewingCurrentLiveVersion = isViewingCurrentLiveCanvasVersion({
    activeCanvasVersionId,
    selectedCanvasVersion,
    effectiveLiveCanvasVersionId,
    liveCanvasVersionId,
  });
  const isEditing = editSessionActive && isViewingCurrentLiveVersion;
  const showLiveActivity = isViewingCurrentLiveVersion && !(editSessionActive && !isRunInspectionMode && !isMemoryMode);

  return {
    activeCanvasVersionId,
    shouldReadStagedCanvasVersionFlag,
    canvasStagingQuery,
    staging,
    loadedCanvasVersion,
    loadedCanvasVersionLoading,
    loadedCanvasVersionFetching,
    awaitingCanvasStaging,
    selectedCanvasVersion,
    isViewingCurrentLiveVersion,
    isViewingLiveVersion: isViewingCurrentLiveVersion,
    isEditing,
    hasEditableVersion: isEditing,
    showLiveActivity,
  };
}
