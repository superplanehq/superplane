import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef } from "react";

import { findPlanningSessionByWorkOrder } from "./planningSessionClient";
import {
  planningSessionHasPendingSurvey,
  planningSessionIsWaiting,
  planningSessionIsWorking,
  type PlanningSessionMachineInput,
  type PlanningSessionPayload,
} from "./planningSessionView";

const SURVEY_POLL_MS = 1500;

export function workOrderPlanningSessionQueryKey(organizationId: string, factoryId: string, workOrderId: string) {
  return ["planning-session-by-work-order", organizationId, factoryId, workOrderId] as const;
}

export function planningActivityPollInterval(
  enabled: boolean,
  session: PlanningSessionPayload | null | undefined,
  backlogAnalyzing: boolean,
): number | false {
  if (!enabled) {
    return false;
  }
  if (planningSessionIsWaiting(session)) {
    return false;
  }
  if (planningSessionIsWorking(session) || backlogAnalyzing) {
    return SURVEY_POLL_MS;
  }
  return false;
}

export type WorkOrderPlanningActivity = {
  hasAgentQuestion: boolean;
  isWaiting: boolean;
  isWorking: boolean;
  /** Session machine fields for `draftCardAgentIsWorking`. Null until enabled. */
  session: PlanningSessionMachineInput | null;
};

/**
 * Live analysis session for a draft card. `enabled` reads the shared
 * cache. Polling continues only while the agent still works.
 */
export function useWorkOrderPlanningActivity(
  organizationId: string,
  factoryId: string,
  workOrderId: string,
  enabled: boolean,
  backlogAnalyzing = false,
): WorkOrderPlanningActivity {
  const queryEnabled = enabled && Boolean(organizationId && factoryId && workOrderId);
  const { data, refetch } = useQuery({
    queryKey: workOrderPlanningSessionQueryKey(organizationId, factoryId, workOrderId),
    queryFn: () => findPlanningSessionByWorkOrder(organizationId, factoryId, workOrderId),
    enabled: queryEnabled,
    refetchInterval: (query) =>
      planningActivityPollInterval(enabled, query.state.data as PlanningSessionPayload | null, backlogAnalyzing),
  });
  const wasBacklogAnalyzing = useRef(backlogAnalyzing);
  useEffect(() => {
    const analysisStopped = wasBacklogAnalyzing.current && !backlogAnalyzing;
    wasBacklogAnalyzing.current = backlogAnalyzing;
    if (queryEnabled && analysisStopped) {
      void refetch();
    }
  }, [backlogAnalyzing, queryEnabled, refetch]);
  if (!enabled) {
    return { hasAgentQuestion: false, isWaiting: false, isWorking: false, session: null };
  }
  return {
    hasAgentQuestion: planningSessionHasPendingSurvey(data),
    isWaiting: planningSessionIsWaiting(data),
    isWorking: planningSessionIsWorking(data),
    session: data ?? null,
  };
}

export function useWorkOrderPlanningSurvey(
  organizationId: string,
  factoryId: string,
  workOrderId: string,
  enabled: boolean,
): boolean {
  return useWorkOrderPlanningActivity(organizationId, factoryId, workOrderId, enabled).hasAgentQuestion;
}
