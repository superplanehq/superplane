import type { CanvasesCanvasRun } from "@/api-client";
import { useMutationState, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";

import { workOrderUpdateMutationKey } from "@/hooks/useFactoryData";

const REFRESH_INTERVAL_MS = 3_000;
const REFRESH_WINDOW_MS = 60_000;

type PlanningRunRestartRequest = {
  attempt: number;
  previousRunId?: string;
};

type BacklogPlanningRunRefreshArgs = {
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  isDraft: boolean;
  hasSession: boolean;
  latestRun?: CanvasesCanvasRun;
};

function awaitsPlanningRun(
  latestRun: CanvasesCanvasRun | undefined,
  restartRequest: PlanningRunRestartRequest | undefined,
) {
  if (restartRequest && latestRun?.id === restartRequest.previousRunId) {
    return true;
  }
  return !latestRun || latestRun.state !== "STATE_FINISHED";
}

/** Resync only while an open draft waits for a run or its retry. */
export function useBacklogPlanningRunRefresh({
  organizationId,
  factoryId,
  orderId,
  isDraft,
  hasSession,
  latestRun,
}: BacklogPlanningRunRefreshArgs): void {
  const queryClient = useQueryClient();
  const modelUpdates = useMutationState({
    filters: {
      mutationKey: workOrderUpdateMutationKey(organizationId ?? "", factoryId ?? ""),
      predicate: (mutation) => {
        const variables = mutation.state.variables;
        return Boolean(
          variables &&
            typeof variables === "object" &&
            "orderId" in variables &&
            variables.orderId === orderId &&
            "refinementModel" in variables,
        );
      },
    },
    select: (mutation) => ({ submittedAt: mutation.state.submittedAt, status: mutation.state.status }),
  });
  const latestModelUpdate = modelUpdates.at(-1);
  const latestModelUpdateAt = latestModelUpdate?.submittedAt;
  const latestModelUpdateStatus = latestModelUpdate?.status;
  const observedModelUpdateAt = useRef(latestModelUpdateAt);
  const [restartRequest, setRestartRequest] = useState<PlanningRunRestartRequest>();

  useEffect(() => {
    if (latestModelUpdateAt === undefined || latestModelUpdateAt === observedModelUpdateAt.current) {
      if (latestModelUpdateStatus === "error") {
        setRestartRequest((current) => (current?.attempt === latestModelUpdateAt ? undefined : current));
      }
      return;
    }
    observedModelUpdateAt.current = latestModelUpdateAt;
    if (latestModelUpdateStatus === "error") {
      setRestartRequest(undefined);
      return;
    }
    setRestartRequest({ attempt: latestModelUpdateAt, previousRunId: latestRun?.id });
  }, [latestModelUpdateAt, latestModelUpdateStatus, latestRun?.id]);

  const shouldRefresh =
    Boolean(organizationId && orderId && isDraft && !hasSession) && awaitsPlanningRun(latestRun, restartRequest);

  useEffect(() => {
    if (!shouldRefresh) {
      return;
    }

    const interval = window.setInterval(() => {
      void queryClient.invalidateQueries({ queryKey: ["backlog-analysis-runs", organizationId] });
    }, REFRESH_INTERVAL_MS);
    const timeout = window.setTimeout(() => window.clearInterval(interval), REFRESH_WINDOW_MS);
    return () => {
      window.clearInterval(interval);
      window.clearTimeout(timeout);
    };
  }, [queryClient, organizationId, orderId, shouldRefresh, latestRun?.id, latestRun?.state, restartRequest?.attempt]);
}
