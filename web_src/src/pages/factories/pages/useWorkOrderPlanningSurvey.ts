import { useQuery } from "@tanstack/react-query";

import { findPlanningSessionByWorkOrder } from "./planningSessionClient";
import {
  draftCardAgentIsWorking,
  planningSessionHasPendingSurvey,
  planningSessionIsWaiting,
  planningSessionIsWorking,
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
): { hasAgentQuestion: boolean; isWaiting: boolean; isWorking: boolean; isAgentWorking: boolean } {
  const { data } = useQuery({
    queryKey: workOrderPlanningSessionQueryKey(organizationId, factoryId, workOrderId),
    queryFn: () => findPlanningSessionByWorkOrder(organizationId, factoryId, workOrderId),
    enabled: enabled && Boolean(organizationId && factoryId && workOrderId),
    refetchInterval: (query) =>
      planningActivityPollInterval(enabled, query.state.data as PlanningSessionPayload | null, backlogAnalyzing),
  });
  if (!enabled) {
    return { hasAgentQuestion: false, isWaiting: false, isWorking: false, isAgentWorking: backlogAnalyzing };
  }
  return {
    hasAgentQuestion: planningSessionHasPendingSurvey(data),
    isWaiting: planningSessionIsWaiting(data),
    isWorking: planningSessionIsWorking(data),
    isAgentWorking: draftCardAgentIsWorking(data, backlogAnalyzing),
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
