import { useCallback, useEffect, useRef, useState } from "react";

export function useRunsDetailState(searchParams: URLSearchParams, isRunsMode: boolean, selectedRunId: string | null) {
  const [openRunDetailOnMount, setOpenRunDetailOnMount] = useState(() => Boolean(searchParams.get("run")));
  const [runDetailNodeId, setRunDetailNodeId] = useState<string | null>(null);
  const [runNodeDetailPaneHeight, setRunNodeDetailPaneHeight] = useState(320);
  const [detailDismissedForRunId, setDetailDismissedForRunId] = useState<string | null>(null);
  const wasRunsModeRef = useRef(isRunsMode);
  const previousSelectedRunIdForDetailRef = useRef<string | null>(selectedRunId);

  useEffect(() => {
    if (!searchParams.get("run")) {
      setDetailDismissedForRunId(null);
    }
  }, [searchParams]);

  useEffect(() => {
    if (isRunsMode && !wasRunsModeRef.current) {
      const runId = searchParams.get("run");
      if (runId && runId !== detailDismissedForRunId) {
        setOpenRunDetailOnMount(true);
      }
    }
    wasRunsModeRef.current = isRunsMode;
  }, [detailDismissedForRunId, isRunsMode, searchParams]);

  useEffect(() => {
    if (previousSelectedRunIdForDetailRef.current === selectedRunId) {
      return;
    }
    previousSelectedRunIdForDetailRef.current = selectedRunId;
    setRunDetailNodeId(null);
  }, [selectedRunId]);

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
