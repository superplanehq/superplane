import { useEffect } from "react";
import { shouldClearStaleRunUrl } from "./workflowPageHelpers";

export function useStaleRunInspectionUrlCleanup({
  selectedRunId,
  isRunInspectionMode,
  selectedRun,
  isRunResolveLoading,
  isRunUnresolvable,
  onClear,
}: {
  selectedRunId: string | null;
  isRunInspectionMode: boolean;
  selectedRun: unknown;
  isRunResolveLoading: boolean;
  isRunUnresolvable: boolean;
  onClear: () => void;
}) {
  useEffect(() => {
    if (
      shouldClearStaleRunUrl({
        selectedRunId,
        isRunInspectionMode,
        selectedRun,
        isRunResolveLoading,
        isRunUnresolvable,
      })
    ) {
      onClear();
    }
  }, [isRunInspectionMode, isRunUnresolvable, isRunResolveLoading, onClear, selectedRun, selectedRunId]);
}
