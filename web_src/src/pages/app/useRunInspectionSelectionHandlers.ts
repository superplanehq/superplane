import { startTransition, useCallback, type Dispatch, type MutableRefObject, type SetStateAction } from "react";
import type { SetURLSearchParams } from "react-router-dom";

import type { CanvasFocusRequest } from "./useAgentNodeFocusRequest";
import { applyRunInspectionNavigationSearchParams, clearRunInspectionSearchParams } from "./viewState";

export type RunSelectionHandlersOptions = {
  hasEditableVersion: boolean;
  liveCanvasVersionId?: string;
  handleUseVersion: (versionId: string, options?: { preserveStagedLayer?: boolean }) => void;
  clearDismissedRunDetail: (options?: { persistAutoOpen?: boolean }) => void;
  setRunDetailNodeId: (value: string | null) => void;
  setFocusRequest: Dispatch<SetStateAction<CanvasFocusRequest | null>>;
  setSearchParams: SetURLSearchParams;
  requestRunFitRef: MutableRefObject<(runId: string) => void>;
  preserveRunDetailNodeOnNextRunChangeRef: MutableRefObject<boolean>;
  searchParams: URLSearchParams;
  runDetailNodeId: string | null;
  clearParticipantFit: () => void;
};

type SharedSelectOptions = Pick<
  RunSelectionHandlersOptions,
  | "clearDismissedRunDetail"
  | "setRunDetailNodeId"
  | "setFocusRequest"
  | "setSearchParams"
  | "requestRunFitRef"
  | "preserveRunDetailNodeOnNextRunChangeRef"
  | "searchParams"
  | "runDetailNodeId"
> & {
  exitEditableVersionForRunInspection: () => void;
};

function useExitEditableVersionForRunInspection(
  hasEditableVersion: boolean,
  liveCanvasVersionId: string | undefined,
  handleUseVersion: RunSelectionHandlersOptions["handleUseVersion"],
) {
  return useCallback(() => {
    if (!hasEditableVersion || !liveCanvasVersionId) {
      return;
    }
    handleUseVersion(liveCanvasVersionId);
  }, [hasEditableVersion, liveCanvasVersionId, handleUseVersion]);
}

function usePrimaryRunSelectHandlers(options: SharedSelectOptions) {
  const {
    exitEditableVersionForRunInspection,
    clearDismissedRunDetail,
    setRunDetailNodeId,
    setFocusRequest,
    setSearchParams,
    requestRunFitRef,
    preserveRunDetailNodeOnNextRunChangeRef,
    searchParams,
  } = options;

  const handleSelectRun = useCallback(
    (runId: string) => {
      exitEditableVersionForRunInspection();
      clearDismissedRunDetail({ persistAutoOpen: true });
      setRunDetailNodeId(null);
      setFocusRequest(null);
      requestRunFitRef.current(runId);
      startTransition(() => {
        setSearchParams((current) => applyRunInspectionNavigationSearchParams(current, { runId }), { replace: true });
      });
    },
    [
      clearDismissedRunDetail,
      exitEditableVersionForRunInspection,
      requestRunFitRef,
      setFocusRequest,
      setRunDetailNodeId,
      setSearchParams,
    ],
  );

  const handleSelectRunFromSidebarEvent = useCallback(
    (runId: string, options?: { nodeId?: string }) => {
      exitEditableVersionForRunInspection();
      clearDismissedRunDetail({ persistAutoOpen: true });
      const inspectorNodeId =
        options?.nodeId ?? (searchParams.get("sidebar") === "1" ? searchParams.get("node") : null);
      if (!inspectorNodeId) requestRunFitRef.current(runId);
      if (inspectorNodeId) {
        preserveRunDetailNodeOnNextRunChangeRef.current = true;
        setRunDetailNodeId(inspectorNodeId);
        setFocusRequest({ nodeId: inspectorNodeId, requestId: Date.now(), targetMode: "runs", tab: "latest" });
      } else {
        setRunDetailNodeId(null);
        setFocusRequest(null);
      }

      setSearchParams(
        (current) =>
          applyRunInspectionNavigationSearchParams(current, {
            runId,
            nodeId: inspectorNodeId,
          }),
        { replace: true },
      );
    },
    [
      clearDismissedRunDetail,
      exitEditableVersionForRunInspection,
      preserveRunDetailNodeOnNextRunChangeRef,
      requestRunFitRef,
      searchParams,
      setFocusRequest,
      setRunDetailNodeId,
      setSearchParams,
    ],
  );

  return { handleSelectRun, handleSelectRunFromSidebarEvent };
}

function useSecondaryRunSelectHandlers(options: SharedSelectOptions) {
  const {
    exitEditableVersionForRunInspection,
    clearDismissedRunDetail,
    setRunDetailNodeId,
    setFocusRequest,
    setSearchParams,
    requestRunFitRef,
    preserveRunDetailNodeOnNextRunChangeRef,
    runDetailNodeId,
  } = options;

  const handleLogRunExecutionSelect = useCallback(
    (options: { runId: string; nodeId: string }) => {
      exitEditableVersionForRunInspection();
      clearDismissedRunDetail({ persistAutoOpen: true });
      preserveRunDetailNodeOnNextRunChangeRef.current = true;
      setRunDetailNodeId(options.nodeId);
      setFocusRequest({ nodeId: options.nodeId, requestId: Date.now(), targetMode: "runs", tab: "latest" });
      setSearchParams(
        (current) =>
          applyRunInspectionNavigationSearchParams(current, {
            runId: options.runId,
            nodeId: options.nodeId,
          }),
        { replace: true },
      );
    },
    [
      clearDismissedRunDetail,
      exitEditableVersionForRunInspection,
      preserveRunDetailNodeOnNextRunChangeRef,
      setFocusRequest,
      setRunDetailNodeId,
      setSearchParams,
    ],
  );

  const handleNavigateRun = useCallback(
    (runId: string) => {
      exitEditableVersionForRunInspection();
      const preservedNodeId = runDetailNodeId;
      preserveRunDetailNodeOnNextRunChangeRef.current = Boolean(preservedNodeId);
      clearDismissedRunDetail({ persistAutoOpen: true });
      setFocusRequest(null);
      requestRunFitRef.current(runId);
      setSearchParams(
        (current) =>
          applyRunInspectionNavigationSearchParams(current, {
            runId,
            nodeId: preservedNodeId,
          }),
        { replace: true },
      );
    },
    [
      clearDismissedRunDetail,
      exitEditableVersionForRunInspection,
      preserveRunDetailNodeOnNextRunChangeRef,
      requestRunFitRef,
      runDetailNodeId,
      setFocusRequest,
      setSearchParams,
    ],
  );

  return { handleLogRunExecutionSelect, handleNavigateRun };
}

function useRunDetailHandlers(
  options: Pick<
    RunSelectionHandlersOptions,
    "clearDismissedRunDetail" | "setRunDetailNodeId" | "setFocusRequest" | "setSearchParams" | "clearParticipantFit"
  >,
) {
  const { clearDismissedRunDetail, setRunDetailNodeId, setFocusRequest, setSearchParams, clearParticipantFit } =
    options;

  const handleClearRunInspection = useCallback(() => {
    setRunDetailNodeId(null);
    setFocusRequest(null);
    clearParticipantFit();
    setSearchParams((current) => clearRunInspectionSearchParams(current), { replace: true });
  }, [clearParticipantFit, setFocusRequest, setRunDetailNodeId, setSearchParams]);

  const handleRunNodeDetailSelection = useCallback(
    (nodeId: string | null) => {
      setRunDetailNodeId(nodeId);
      if (nodeId) {
        clearDismissedRunDetail({ persistAutoOpen: true });
      } else {
        setFocusRequest(null);
      }

      setSearchParams(
        (current) => {
          const next = new URLSearchParams(current);
          if (nodeId) {
            next.set("sidebar", "1");
            next.set("node", nodeId);
          } else {
            next.delete("sidebar");
            next.delete("node");
          }
          return next;
        },
        { replace: true },
      );
    },
    [clearDismissedRunDetail, setFocusRequest, setRunDetailNodeId, setSearchParams],
  );

  const handleRunNodeDetailNavigate = useCallback(
    (nodeId: string) => {
      handleRunNodeDetailSelection(nodeId);
      setFocusRequest({ nodeId, requestId: Date.now(), targetMode: "runs", tab: "latest" });
    },
    [handleRunNodeDetailSelection, setFocusRequest],
  );

  const handleSelectLiveCanvas = useCallback(() => {
    handleClearRunInspection();
  }, [handleClearRunInspection]);

  return {
    handleClearRunInspection,
    handleSelectLiveCanvas,
    handleRunNodeDetailSelection,
    handleRunNodeDetailNavigate,
  };
}

export function useRunInspectionSelectionHandlers(options: RunSelectionHandlersOptions) {
  const exitEditableVersionForRunInspection = useExitEditableVersionForRunInspection(
    options.hasEditableVersion,
    options.liveCanvasVersionId,
    options.handleUseVersion,
  );
  const shared = { ...options, exitEditableVersionForRunInspection };
  const primary = usePrimaryRunSelectHandlers(shared);
  const secondary = useSecondaryRunSelectHandlers(shared);
  const detailHandlers = useRunDetailHandlers(options);

  return {
    exitEditableVersionForRunInspection,
    ...primary,
    ...secondary,
    ...detailHandlers,
  };
}
