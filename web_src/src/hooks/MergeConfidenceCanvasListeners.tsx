import type { CanvasesCanvasRun } from "@/api-client";
import { useCanvasWebsocket } from "@/hooks/useCanvasWebsocket";
import { upsertMergeConfidenceCanvasRun } from "@/pages/factories/lib/mergeConfidenceRuns";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";

import { mergeConfidenceRunsKey } from "./useMergeConfidenceRuns";

export function MergeConfidenceCanvasListeners({
  organizationId,
  canvasIds,
}: {
  organizationId: string;
  canvasIds: string[];
}) {
  return canvasIds.map((canvasId) => (
    <MergeConfidenceCanvasListener key={canvasId} organizationId={organizationId} canvasId={canvasId} />
  ));
}

function MergeConfidenceCanvasListener({ organizationId, canvasId }: { organizationId: string; canvasId: string }) {
  const queryClient = useQueryClient();
  const queryKey = useMemo(() => mergeConfidenceRunsKey(organizationId, canvasId), [canvasId, organizationId]);
  const applyRunEvent = useCallback(
    (run: CanvasesCanvasRun) => {
      queryClient.setQueryData<CanvasesCanvasRun[]>(queryKey, (current) =>
        upsertMergeConfidenceCanvasRun(current, run),
      );
    },
    [queryClient, queryKey],
  );
  const resynchronizeRuns = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey });
  }, [queryClient, queryKey]);

  useCanvasWebsocket({
    canvasId,
    organizationId,
    processRuntimeEvents: false,
    enabled: Boolean(organizationId && canvasId),
    onRunEvent: applyRunEvent,
    onConnectionOpen: resynchronizeRuns,
  });

  return null;
}
