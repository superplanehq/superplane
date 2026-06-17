import { useEffect } from "react";
import { shouldClearStaleRunUrl } from "./workflowPageHelpers";

export function useStaleRunInspectionUrlCleanup({
  selectedRunId,
  isRunInspectionMode,
  selectedRun,
  isRunResolveLoading,
  describeRunSettled,
  onClear,
}: {
  selectedRunId: string | null;
  isRunInspectionMode: boolean;
  selectedRun: unknown;
  isRunResolveLoading: boolean;
  describeRunSettled: boolean;
  onClear: () => void;
}) {
  useEffect(() => {
    if (
      shouldClearStaleRunUrl({
        selectedRunId,
        isRunInspectionMode,
        selectedRun,
        isRunResolveLoading,
        describeRunSettled,
      })
    ) {
      onClear();
    }
  }, [describeRunSettled, isRunInspectionMode, isRunResolveLoading, onClear, selectedRun, selectedRunId]);
}
