import { useMemo } from "react";
import type { CanvasesCanvas, CanvasesCanvasRun } from "@/api-client";
import { useCanvasVersion } from "@/hooks/useCanvasData";

type UseSelectedRunCanvasParams = {
  organizationId: string;
  canvasId: string;
  selectedRun: CanvasesCanvasRun | null;
  isRunsMode: boolean;
  liveCanvasVersionId?: string;
  canvas?: CanvasesCanvas | null;
  liveCanvas?: CanvasesCanvas | null;
};

export function useSelectedRunCanvas({
  organizationId,
  canvasId,
  selectedRun,
  isRunsMode,
  liveCanvasVersionId,
  canvas,
  liveCanvas,
}: UseSelectedRunCanvasParams) {
  const selectedRunVersionId = selectedRun?.versionId || "";
  const selectedRunVersionQuery = useCanvasVersion(
    organizationId,
    canvasId,
    selectedRunVersionId,
    isRunsMode && !!selectedRunVersionId && selectedRunVersionId !== liveCanvasVersionId,
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
      return canvas;
    }

    return {
      ...liveCanvas,
      spec: selectedRunVersion.spec,
    };
  }, [canvas, liveCanvas, liveCanvasVersionId, selectedRunVersion?.spec, selectedRunVersionId]);

  return {
    selectedRunCanvas,
    isSelectedRunVersionLoading:
      isRunsMode &&
      !!selectedRunVersionId &&
      selectedRunVersionId !== liveCanvasVersionId &&
      !selectedRunVersion?.spec &&
      !selectedRunVersionQuery.isError &&
      (selectedRunVersionQuery.isLoading || selectedRunVersionQuery.isFetching),
  };
}
