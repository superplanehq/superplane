import { useEffect } from "react";
import { shouldClearStaleRunUrl } from "./runInspectionSync";

export function useStaleRunInspectionUrlCleanup({
  selectedRunId,
  isRunInspectionMode,
  selectedRun,
  isRunResolveLoading,
  isRunNotFound,
  onClear,
}: {
  selectedRunId: string | null;
  isRunInspectionMode: boolean;
  selectedRun: unknown;
  isRunResolveLoading: boolean;
  isRunNotFound: boolean;
  onClear: () => void;
}) {
  useEffect(() => {
    if (
      shouldClearStaleRunUrl({
        selectedRunId,
        isRunInspectionMode,
        selectedRun,
        isRunResolveLoading,
        isRunNotFound,
      })
    ) {
      onClear();
    }
  }, [isRunInspectionMode, isRunNotFound, isRunResolveLoading, onClear, selectedRun, selectedRunId]);
}
