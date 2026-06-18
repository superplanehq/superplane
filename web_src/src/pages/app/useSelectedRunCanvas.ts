import { useMemo } from "react";
import type { CanvasesCanvas, CanvasesCanvasRun } from "@/api-client";
import { useCanvasVersion } from "@/hooks/useCanvasData";

type UseSelectedRunCanvasParams = {
  organizationId: string;
  canvasId: string;
  selectedRun: CanvasesCanvasRun | null;
  isRunInspectionMode: boolean;
  liveCanvasVersionId?: string;
  canvas?: CanvasesCanvas | null;
  liveCanvas?: CanvasesCanvas | null;
};

export function useSelectedRunCanvas({
  organizationId,
  canvasId,
  selectedRun,
  isRunInspectionMode,
  liveCanvasVersionId,
  canvas,
  liveCanvas,
}: UseSelectedRunCanvasParams) {
  const selectedRunVersionId = selectedRun?.versionId || "";
  const selectedRunVersionQuery = useCanvasVersion(
    organizationId,
    canvasId,
    selectedRunVersionId,
    isRunInspectionMode && !!selectedRunVersionId && selectedRunVersionId !== liveCanvasVersionId,
  );
  const selectedRunVersion = selectedRunVersionQuery.data;

  const selectedRunCanvas = useMemo(() => {
    if (!selectedRunVersionId) {
      return canvas;
    }

    if (selectedRunVersionId === liveCanvasVersionId) {
      return liveCanvas || canvas;
    }

    if (!liveCanvas || !selectedRunVersion?.spec) {
      return null;
    }

    return {
      ...liveCanvas,
      spec: selectedRunVersion.spec,
    };
  }, [canvas, liveCanvas, liveCanvasVersionId, selectedRunVersion?.spec, selectedRunVersionId]);

  return {
    selectedRunCanvas,
    isSelectedRunVersionLoading:
      isRunInspectionMode &&
      !!selectedRunVersionId &&
      selectedRunVersionId !== liveCanvasVersionId &&
      !selectedRunVersion?.spec &&
      !selectedRunVersionQuery.isError &&
      (selectedRunVersionQuery.isLoading || selectedRunVersionQuery.isFetching),
  };
}
