import { useCallback, useEffect, useMemo, useRef, useState, type RefObject } from "react";

function getRunDetailNodeIdFromSearchParams(
  searchParams: URLSearchParams,
  isRunInspectionMode: boolean,
  selectedRunId: string | null,
): string | null {
  const runId = searchParams.get("run");
  if (!isRunInspectionMode || !runId || runId !== selectedRunId || searchParams.get("sidebar") !== "1") {
    return null;
  }

  return searchParams.get("node") || null;
}

export function useRunsDetailState(
  searchParams: URLSearchParams,
  isRunInspectionMode: boolean,
  selectedRunId: string | null,
  preserveRunDetailNodeOnNextRunChangeRef?: RefObject<boolean>,
  onBackToRunList?: () => void,
) {
  const [openRunDetailOnMount, setOpenRunDetailOnMount] = useState(() => Boolean(searchParams.get("run")));
  const urlRunDetailNodeId = useMemo(
    () => getRunDetailNodeIdFromSearchParams(searchParams, isRunInspectionMode, selectedRunId),
    [isRunInspectionMode, searchParams, selectedRunId],
  );
  const [runDetailNodeId, setRunDetailNodeId] = useState<string | null>(() =>
    getRunDetailNodeIdFromSearchParams(searchParams, isRunInspectionMode, selectedRunId),
  );
  const [runNodeDetailPaneHeight, setRunNodeDetailPaneHeight] = useState(320);
  const [detailDismissedForRunId, setDetailDismissedForRunId] = useState<string | null>(null);
  const wasRunInspectionModeRef = useRef(isRunInspectionMode);
  const previousSelectedRunIdForDetailRef = useRef<string | null>(selectedRunId);
  const previousUrlRunDetailNodeIdRef = useRef<string | null>(urlRunDetailNodeId);

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
      previousUrlRunDetailNodeIdRef.current = urlRunDetailNodeId;
      return;
    }
    setRunDetailNodeId(urlRunDetailNodeId);
  }, [preserveRunDetailNodeOnNextRunChangeRef, selectedRunId, urlRunDetailNodeId]);

  useEffect(() => {
    if (previousUrlRunDetailNodeIdRef.current === urlRunDetailNodeId) {
      return;
    }
    previousUrlRunDetailNodeIdRef.current = urlRunDetailNodeId;
    setRunDetailNodeId(urlRunDetailNodeId);
  }, [urlRunDetailNodeId]);

  const clearDismissedRunDetail = useCallback(() => {
    setDetailDismissedForRunId(null);
  }, []);

  const handleBackToRunList = useCallback(() => {
    setDetailDismissedForRunId(selectedRunId);
    setRunDetailNodeId(null);
    setOpenRunDetailOnMount(false);
    onBackToRunList?.();
  }, [onBackToRunList, selectedRunId]);

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
