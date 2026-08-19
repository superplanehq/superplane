import { useEffect, useRef, type MutableRefObject } from "react";
import { shouldRequestInitialRunFit } from "./workflowPageHelpers";

/**
 * Requests a "fit to participants" once, on mount, when run-inspection mode
 * is already active from the URL (e.g. a Lines/Automations/Work Order deep
 * link landing directly on `?run=<id>`). Mirrors what `handleSelectRun`
 * already does for in-app run selection, closing the gap where a fresh
 * mount into run-inspection mode never requests a fit at all.
 *
 * Fires at most once per mount: subsequent run changes during the same
 * mount are already handled explicitly by `handleSelectRun`,
 * `handleNavigateRun`, and `handleSelectRunFromSidebarEvent`.
 */
export function useInitialRunFitOnEntry({
  isRunInspectionMode,
  selectedRunId,
  searchParams,
  requestRunFitRef,
}: {
  isRunInspectionMode: boolean;
  selectedRunId: string | null;
  searchParams: URLSearchParams;
  requestRunFitRef: MutableRefObject<(runId: string) => void>;
}) {
  const handledRef = useRef(false);

  useEffect(() => {
    if (handledRef.current) return;
    if (!shouldRequestInitialRunFit({ isRunInspectionMode, selectedRunId, searchParams })) return;
    handledRef.current = true;
    requestRunFitRef.current(selectedRunId!);
  }, [isRunInspectionMode, requestRunFitRef, searchParams, selectedRunId]);
}
