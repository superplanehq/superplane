import { useCallback, useEffect, useRef, useState, type RefObject } from "react";

function getInitialRunDetailNodeId(searchParams: URLSearchParams, isRunInspectionMode: boolean): string | null {
  if (!isRunInspectionMode || !searchParams.get("run") || searchParams.get("sidebar") !== "1") {
    return null;
  }

  return searchParams.get("node") || null;
}

export function useRunsDetailState(
  searchParams: URLSearchParams,
  isRunInspectionMode: boolean,
  selectedRunId: string | null,
  preserveRunDetailNodeOnNextRunChangeRef?: RefObject<boolean>,
) {
  const [openRunDetailOnMount, setOpenRunDetailOnMount] = useState(() => Boolean(searchParams.get("run")));
  const [runDetailNodeId, setRunDetailNodeId] = useState<string | null>(() =>
    getInitialRunDetailNodeId(searchParams, isRunInspectionMode),
  );
  const [runNodeDetailPaneHeight, setRunNodeDetailPaneHeight] = useState(320);
  const [detailDismissedForRunId, setDetailDismissedForRunId] = useState<string | null>(null);
  const wasRunInspectionModeRef = useRef(isRunInspectionMode);
  const previousSelectedRunIdForDetailRef = useRef<string | null>(selectedRunId);

  useEffect(() => {
    if (!searchParams.get("run")) {
      setDetailDismissedForRunId(null);
    }
  }, [searchParams]);

  useEffect(() => {
    if (isRunInspectionMode && !wasRunInspectionModeRef.current) {
      const runId = searchParams.get("run");
      setOpenRunDetailOnMount(Boolean(runId && runId !== detailDismissedForRunId));
    } else if (!isRunInspectionMode && wasRunInspectionModeRef.current) {
      setOpenRunDetailOnMount(false);
    }
    wasRunInspectionModeRef.current = isRunInspectionMode;
  }, [detailDismissedForRunId, isRunInspectionMode, searchParams]);

  useEffect(() => {
    if (previousSelectedRunIdForDetailRef.current === selectedRunId) {
      return;
    }
    previousSelectedRunIdForDetailRef.current = selectedRunId;
    if (preserveRunDetailNodeOnNextRunChangeRef?.current) {
      preserveRunDetailNodeOnNextRunChangeRef.current = false;
      return;
    }
    setRunDetailNodeId(null);
  }, [preserveRunDetailNodeOnNextRunChangeRef, selectedRunId]);

  const clearDismissedRunDetail = useCallback(() => {
    setDetailDismissedForRunId(null);
  }, []);

  const handleBackToRunList = useCallback(() => {
    setDetailDismissedForRunId(selectedRunId);
    setRunDetailNodeId(null);
    setOpenRunDetailOnMount(false);
  }, [selectedRunId]);

  return {
    openRunDetailOnMount,
    runDetailNodeId,
    setRunDetailNodeId,
    runNodeDetailPaneHeight,
    setRunNodeDetailPaneHeight,
    clearDismissedRunDetail,
    detailDismissedForRunId,
    handleBackToRunList,
  };
}
