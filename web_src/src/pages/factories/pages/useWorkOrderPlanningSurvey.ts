import { useQuery } from "@tanstack/react-query";

import { findPlanningSessionByWorkOrder } from "./planningSessionClient";
import { planningSessionHasPendingSurvey } from "./planningSessionView";

const SURVEY_POLL_MS = 1500;

export function workOrderPlanningSessionQueryKey(organizationId: string, factoryId: string, workOrderId: string) {
  return ["planning-session-by-work-order", organizationId, factoryId, workOrderId] as const;
}

/**
 * True when the analysis session for this task has an unanswered
 * multiple-choice question. Polls only while `enabled` is true.
 */
export function useWorkOrderPlanningSurvey(
  organizationId: string,
  factoryId: string,
  workOrderId: string,
  enabled: boolean,
): boolean {
  const { data } = useQuery({
    queryKey: workOrderPlanningSessionQueryKey(organizationId, factoryId, workOrderId),
    queryFn: () => findPlanningSessionByWorkOrder(organizationId, factoryId, workOrderId),
    enabled: enabled && Boolean(organizationId && factoryId && workOrderId),
    refetchInterval: enabled ? SURVEY_POLL_MS : false,
  });
  return planningSessionHasPendingSurvey(data);
}
