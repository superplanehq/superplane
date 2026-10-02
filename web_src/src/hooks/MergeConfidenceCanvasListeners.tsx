import type { CanvasesCanvasRun, FactoriesFactoryPullRequest } from "@/api-client";
import { useCanvasWebsocket } from "@/hooks/useCanvasWebsocket";
import {
  mergeConfidenceRunsForPullRequests,
  upsertMergeConfidenceCanvasRun,
} from "@/pages/factories/lib/mergeConfidenceRuns";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";

import {
  mergeConfidenceDescribedRunKey,
  mergeConfidenceRunsKey,
  mergeConfidenceScoredRunRefsPrefix,
} from "./useMergeConfidenceRuns";

export function MergeConfidenceCanvasListeners({
  organizationId,
  factoryId,
  orderId,
  canvasIds,
  taskKey,
  pullRequests,
}: {
  organizationId: string;
  factoryId: string;
  orderId: string;
  canvasIds: string[];
  taskKey: string;
  pullRequests: FactoriesFactoryPullRequest[];
}) {
  return canvasIds.map((canvasId) => (
    <MergeConfidenceCanvasListener
      key={canvasId}
      organizationId={organizationId}
      factoryId={factoryId}
      orderId={orderId}
      canvasId={canvasId}
      taskKey={taskKey}
      pullRequests={pullRequests}
    />
  ));
}

function MergeConfidenceCanvasListener({
  organizationId,
  factoryId,
  orderId,
  canvasId,
  taskKey,
  pullRequests,
}: {
  organizationId: string;
  factoryId: string;
  orderId: string;
  canvasId: string;
  taskKey: string;
  pullRequests: FactoriesFactoryPullRequest[];
}) {
  const queryClient = useQueryClient();
  const queryKey = useMemo(
    () => mergeConfidenceRunsKey(organizationId, canvasId, taskKey),
    [canvasId, organizationId, taskKey],
  );
  const applyRunEvent = useCallback(
    (run: CanvasesCanvasRun) => {
      if (mergeConfidenceRunsForPullRequests({ id: canvasId, name: "" }, [run], pullRequests).length === 0) {
        return;
      }
      queryClient.setQueriesData<CanvasesCanvasRun[]>({ queryKey }, (current) =>
        upsertMergeConfidenceCanvasRun(current, run),
      );
      if (run.id) {
        queryClient.setQueryData(mergeConfidenceDescribedRunKey(organizationId, canvasId, run.id), run);
      }
    },
    [canvasId, organizationId, pullRequests, queryClient, queryKey],
  );
  const resynchronizeRuns = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey });
    void queryClient.invalidateQueries({
      queryKey: ["merge-confidence-described-run", organizationId, canvasId],
    });
    void queryClient.invalidateQueries({
      queryKey: mergeConfidenceScoredRunRefsPrefix(organizationId, factoryId, orderId),
    });
  }, [canvasId, factoryId, orderId, organizationId, queryClient, queryKey]);

  useCanvasWebsocket({
    canvasId,
    organizationId,
    processRuntimeEvents: false,
    enabled: Boolean(organizationId && canvasId && taskKey),
    onRunEvent: applyRunEvent,
    onConnectionOpen: resynchronizeRuns,
  });

  return null;
}
