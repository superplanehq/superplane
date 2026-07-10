import { useCallback, useEffect, useMemo, useRef, useState, type RefObject } from "react";

const RUN_INSPECTOR_AUTO_OPEN_STORAGE_KEY = "superplane.runInspector.autoOpen";

export function runInspectorAutoOpenStorageKey(canvasId?: string): string {
  return canvasId ? `${RUN_INSPECTOR_AUTO_OPEN_STORAGE_KEY}:${canvasId}` : RUN_INSPECTOR_AUTO_OPEN_STORAGE_KEY;
}

function readRunInspectorAutoOpen(canvasId?: string): boolean {
  if (typeof window === "undefined") return true;

  try {
    const stored = window.localStorage.getItem(runInspectorAutoOpenStorageKey(canvasId));
    if (stored === null) return true;
    return stored === "true";
  } catch {
    return true;
  }
}

function writeRunInspectorAutoOpen(canvasId: string | undefined, open: boolean) {
  if (typeof window === "undefined") return;

  try {
    window.localStorage.setItem(runInspectorAutoOpenStorageKey(canvasId), String(open));
  } catch {
    // Ignore storage failures; the in-memory state still reflects the click.
  }
}

type ClearDismissedRunDetailOptions = {
  persistAutoOpen?: boolean;
};

type UseRunsDetailStateOptions = {
  canvasId?: string;
  onBackToRunList?: () => void;
};

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
  options: UseRunsDetailStateOptions = {},
) {
  const { canvasId, onBackToRunList } = options;
  const [openRunDetailOnMount, setOpenRunDetailOnMount] = useState(() => Boolean(searchParams.get("run")));
  const [autoOpenRunDetail, setAutoOpenRunDetail] = useState(() => readRunInspectorAutoOpen(canvasId));
  const urlRunDetailNodeId = useMemo(
    () => getRunDetailNodeIdFromSearchParams(searchParams, isRunInspectionMode, selectedRunId),
    [isRunInspectionMode, searchParams, selectedRunId],
  );
  const [runDetailNodeId, setRunDetailNodeId] = useState<string | null>(() =>
    getRunDetailNodeIdFromSearchParams(searchParams, isRunInspectionMode, selectedRunId),
  );
  const [detailDismissedForRunId, setDetailDismissedForRunId] = useState<string | null>(null);
  const wasRunInspectionModeRef = useRef(isRunInspectionMode);
  const previousSelectedRunIdForDetailRef = useRef<string | null>(selectedRunId);
  const previousUrlRunDetailNodeIdRef = useRef<string | null>(urlRunDetailNodeId);
  const previousCanvasIdRef = useRef<string | undefined>(canvasId);

  useEffect(() => {
    if (previousCanvasIdRef.current === canvasId) return;

    previousCanvasIdRef.current = canvasId;
    setAutoOpenRunDetail(readRunInspectorAutoOpen(canvasId));
  }, [canvasId]);

  useEffect(() => {
    if (!searchParams.get("run")) {
      setDetailDismissedForRunId(null);
    }
  }, [searchParams]);

  useEffect(() => {
    const urlRequestsRunDetail =
      isRunInspectionMode && searchParams.get("run") === selectedRunId && searchParams.get("sidebar") === "1";
    if (urlRequestsRunDetail) {
      setDetailDismissedForRunId(null);
    }
  }, [isRunInspectionMode, searchParams, selectedRunId]);

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

  const setRunDetailAutoOpen = useCallback(
    (open: boolean) => {
      setAutoOpenRunDetail(open);
      writeRunInspectorAutoOpen(canvasId, open);
    },
    [canvasId],
  );

  const clearDismissedRunDetail = useCallback(
    (options?: ClearDismissedRunDetailOptions) => {
      setDetailDismissedForRunId(null);
      if (options?.persistAutoOpen) {
        setRunDetailAutoOpen(true);
      }
    },
    [setRunDetailAutoOpen],
  );

  const maybeOpenRunDetailForRun = useCallback(
    (runId: string | null) => {
      setDetailDismissedForRunId(autoOpenRunDetail ? null : runId);
    },
    [autoOpenRunDetail],
  );

  const handleBackToRunList = useCallback(() => {
    setDetailDismissedForRunId(selectedRunId);
    setRunDetailAutoOpen(false);
    setRunDetailNodeId(null);
    setOpenRunDetailOnMount(false);
    onBackToRunList?.();
  }, [onBackToRunList, selectedRunId, setRunDetailAutoOpen]);

  return {
    openRunDetailOnMount,
    runDetailNodeId,
    setRunDetailNodeId,
    clearDismissedRunDetail,
    maybeOpenRunDetailForRun,
    detailDismissedForRunId,
    handleBackToRunList,
  };
}
