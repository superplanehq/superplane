import type {
  CanvasesCanvasRun,
  CanvasesListRunsResponse,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { useSidebarEventRunLookup } from "@/hooks/useSidebarEventRunLookup";
import type { QueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, type Dispatch, type MutableRefObject, type SetStateAction } from "react";
import type { SetURLSearchParams } from "react-router-dom";

import type { CanvasFocusRequest } from "./useAgentNodeFocusRequest";
import type { RunCanvasData } from "./useRunCanvasData";
import { useLiveCanvasRunInspectionClicks } from "./useLiveCanvasRunInspectionClicks";
import { useRunInspectionSelectionHandlers } from "./useRunInspectionSelectionHandlers";
import { useStaleRunInspectionUrlCleanup } from "./useStaleRunInspectionUrlCleanup";

type UseRunInspectionNavigationOptions = {
  hasEditableVersion: boolean;
  liveCanvasVersionId?: string;
  handleUseVersion: (versionId: string, options?: { preserveStagedLayer?: boolean }) => void;
  isViewingLiveVersion: boolean;
  isEditing: boolean;
  isRunInspectionMode: boolean;
  clearDismissedRunDetail: (options?: { persistAutoOpen?: boolean }) => void;
  setRunDetailNodeId: (value: string | null) => void;
  setFocusRequest: Dispatch<SetStateAction<CanvasFocusRequest | null>>;
  setSearchParams: SetURLSearchParams;
  requestRunFitRef: MutableRefObject<(runId: string) => void>;
  preserveRunDetailNodeOnNextRunChangeRef: MutableRefObject<boolean>;
  searchParams: URLSearchParams;
  runDetailNodeId: string | null;
  clearParticipantFit: () => void;
  selectedRunId: string | null;
  selectedRun: CanvasesCanvasRun | null;
  isSelectedRunLoading: boolean;
  describeRunSettled: boolean;
  canvasId?: string;
  organizationId?: string | null;
  queryClient: QueryClient;
  runs: CanvasesCanvasRun[];
  infiniteRunsPages?: Array<CanvasesListRunsResponse | undefined>;
  runCanvasData: RunCanvasData | null;
  canvasNodesById: Map<string, ComponentsNode>;
  refetchNodeDataMethod: (
    workflowId: string,
    nodeId: string,
    nodeType: string,
    queryClient: QueryClient,
  ) => Promise<void>;
};

/**
 * Run-inspection navigation: select/clear runs, node detail, and live-canvas
 * click → latest-run lookup — kept out of AppPage for eslint line budget.
 */
export function useRunInspectionNavigation(options: UseRunInspectionNavigationOptions) {
  const {
    isViewingLiveVersion,
    isEditing,
    isRunInspectionMode,
    selectedRunId,
    selectedRun,
    isSelectedRunLoading,
    describeRunSettled,
    canvasId,
    organizationId,
    queryClient,
    runs,
    infiniteRunsPages,
    runCanvasData,
    canvasNodesById,
    refetchNodeDataMethod,
    hasEditableVersion,
    liveCanvasVersionId,
    handleUseVersion,
  } = options;

  const selection = useRunInspectionSelectionHandlers(options);
  const runLookupEnabled = isViewingLiveVersion && !isEditing;
  const liveSidebarRunLookupEnabled = runLookupEnabled && !isRunInspectionMode;

  const { resolveRunIdForSidebarEvent, fetchRunIdForSidebarEvent } = useSidebarEventRunLookup({
    enabled: runLookupEnabled,
    canvasId,
    organizationId,
    queryClient,
    runs,
    infiniteRunsPages,
  });

  useStaleRunInspectionUrlCleanup({
    selectedRunId,
    isRunInspectionMode,
    selectedRun,
    isRunResolveLoading: isSelectedRunLoading,
    describeRunSettled,
    onClear: selection.handleClearRunInspection,
  });

  const {
    handleSelectRun,
    handleSelectRunFromSidebarEvent,
    handleLogRunExecutionSelect,
    handleNavigateRun,
    handleClearRunInspection,
    handleSelectLiveCanvas,
    handleRunNodeDetailSelection,
    handleRunNodeDetailNavigate,
  } = selection;

  const handleRunCanvasNodeClick = useCallback(
    (nodeId: string) => {
      if (!isRunInspectionMode || !selectedRun) return;
      const participants = runCanvasData?.participantNodeIds;
      if (participants && participants.length > 0 && !participants.includes(nodeId)) {
        return;
      }
      handleRunNodeDetailSelection(nodeId);
    },
    [handleRunNodeDetailSelection, isRunInspectionMode, runCanvasData, selectedRun],
  );

  const { handleLiveCanvasNodeClick } = useLiveCanvasRunInspectionClicks({
    isRunInspectionMode,
    isEditing,
    liveSidebarRunLookupEnabled,
    canvasId,
    canvasNodesById,
    queryClient,
    refetchNodeDataMethod,
    resolveRunIdForSidebarEvent,
    fetchRunIdForSidebarEvent,
    handleSelectRunFromSidebarEvent,
  });

  useEffect(() => {
    if (!isRunInspectionMode || isViewingLiveVersion) return;
    // Entering an edit session on a draft exits run inspection rather than
    // snapping back to the live version (which would bounce the user out of edit
    // mode). For non-editable previews, keep pinning run inspection to live.
    if (hasEditableVersion) {
      handleClearRunInspection();
      return;
    }
    if (!liveCanvasVersionId) return;
    handleUseVersion(liveCanvasVersionId);
  }, [
    hasEditableVersion,
    handleClearRunInspection,
    handleUseVersion,
    isRunInspectionMode,
    isViewingLiveVersion,
    liveCanvasVersionId,
  ]);

  return {
    runLookupEnabled,
    liveSidebarRunLookupEnabled,
    resolveRunIdForSidebarEvent,
    fetchRunIdForSidebarEvent,
    handleSelectRun,
    handleSelectRunFromSidebarEvent,
    handleLogRunExecutionSelect,
    handleNavigateRun,
    handleClearRunInspection,
    handleSelectLiveCanvas,
    handleRunNodeDetailSelection,
    handleRunNodeDetailNavigate,
    handleRunCanvasNodeClick,
    handleLiveCanvasNodeClick,
  };
}
