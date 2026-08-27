import type { FactoriesFactoryPrFeedbackHandler, FactoriesFactoryPrFeedbackHandlerRun } from "@/api-client";
import {
  factoryPRFeedbackHandlerRunsKey,
  fetchFactoryPRFeedbackHandlerRuns,
  useFactoryPRFeedbackHandlers,
} from "@/hooks/useFactoryPRFeedbackData";
import { useQueries } from "@tanstack/react-query";

import {
  activePRFeedbackWorkOrderIds,
  isActivePRFeedbackRunStatus,
  matchPRFeedbackRunsForWorkOrder,
  type PRFeedbackLogRun,
  type PRFeedbackRunMatch,
} from "./prFeedbackSettingsModel";

function useFactoryPRFeedbackRunLists(organizationId: string, factoryId: string, enabled = true) {
  const handlersQuery = useFactoryPRFeedbackHandlers(organizationId, factoryId);
  const handlers = handlersQuery.data ?? [];
  const runQueries = useQueries({
    queries: handlers.map((handler) => ({
      queryKey: factoryPRFeedbackHandlerRunsKey(organizationId, factoryId, handler.id ?? ""),
      queryFn: () => fetchFactoryPRFeedbackHandlerRuns(organizationId, factoryId, handler.id ?? ""),
      enabled: Boolean(organizationId && factoryId && handler.id) && enabled,
      refetchInterval: 10_000,
    })),
  });
  return { handlers, runsByHandler: runQueries.map((query) => query.data ?? []) };
}

export function useActivePRFeedbackWorkOrderIds(organizationId: string, factoryId: string): ReadonlySet<string> {
  const { runsByHandler } = useFactoryPRFeedbackRunLists(organizationId, factoryId);
  return activePRFeedbackWorkOrderIds(runsByHandler);
}

export function useWorkOrderPRFeedbackLog(
  organizationId: string,
  factoryId: string,
  workOrderId: string | undefined,
): PRFeedbackLogRun[] {
  const { handlers, runsByHandler } = useFactoryPRFeedbackRunLists(
    organizationId,
    factoryId,
    Boolean(organizationId && factoryId && workOrderId),
  );
  if (!workOrderId) {
    return [];
  }
  return matchPRFeedbackRunsForWorkOrder(handlers, runsByHandler, workOrderId).flatMap(splitRunPRFeedbackRunFromMatch);
}

export function matchOldestActivePRFeedbackRun(
  handlers: FactoriesFactoryPrFeedbackHandler[],
  runsByHandler: FactoriesFactoryPrFeedbackHandlerRun[][],
  workOrderId: string,
): PRFeedbackRunMatch | undefined {
  return matchPRFeedbackRunsForWorkOrder(handlers, runsByHandler, workOrderId).find((match) =>
    isActivePRFeedbackRunStatus(match.run.status),
  );
}

function splitRunPRFeedbackRunFromMatch(match: PRFeedbackRunMatch): PRFeedbackLogRun[] {
  if (!match.handler.canvasId || !match.run.id) {
    return [];
  }
  return [{ canvasId: match.handler.canvasId, handlerName: match.handler.name, run: match.run }];
}
