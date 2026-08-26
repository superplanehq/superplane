import type { FactoriesFactoryPrFeedbackHandler, FactoriesFactoryPrFeedbackHandlerRun } from "@/api-client";
import {
  factoryPRFeedbackHandlerRunsKey,
  fetchFactoryPRFeedbackHandlerRuns,
  useFactoryPRFeedbackHandlers,
} from "@/hooks/useFactoryPRFeedbackData";
import { useQueries } from "@tanstack/react-query";

import { factoryAppRunPath } from "../lib/factoryPagePaths";
import { oldestActivePRFeedbackRun } from "./prFeedbackSettingsModel";

export function useWorkOrderPRFeedbackRunHref(
  organizationId: string,
  factoryId: string,
  factoryKey: string,
  workOrderId: string | undefined,
): string | undefined {
  const handlersQuery = useFactoryPRFeedbackHandlers(organizationId, factoryId);
  const handlers = handlersQuery.data ?? [];
  const runQueries = useQueries({
    queries: handlers.map((handler) => ({
      queryKey: factoryPRFeedbackHandlerRunsKey(organizationId, factoryId, handler.id ?? ""),
      queryFn: () => fetchFactoryPRFeedbackHandlerRuns(organizationId, factoryId, handler.id ?? ""),
      enabled: Boolean(organizationId && factoryId && handler.id && workOrderId),
      refetchInterval: 10_000,
    })),
  });

  if (!workOrderId) {
    return undefined;
  }

  const match = matchOldestActivePRFeedbackRun(
    handlers,
    runQueries.map((query) => query.data ?? []),
    workOrderId,
  );
  if (!match?.run.id || !match.handler.canvasId) {
    return undefined;
  }

  return factoryAppRunPath(organizationId, factoryKey, match.handler.canvasId, match.run.id, { from: "work-order" });
}

export function matchOldestActivePRFeedbackRun(
  handlers: FactoriesFactoryPrFeedbackHandler[],
  runsByHandler: FactoriesFactoryPrFeedbackHandlerRun[][],
  workOrderId: string,
): { handler: FactoriesFactoryPrFeedbackHandler; run: FactoriesFactoryPrFeedbackHandlerRun } | undefined {
  const candidates = handlers.flatMap((handler, index) => {
    const run = oldestActivePRFeedbackRun(runsByHandler[index] ?? [], workOrderId);
    return run ? [{ handler, run }] : [];
  });
  if (candidates.length === 0) {
    return undefined;
  }
  return [...candidates].sort(
    (left, right) => Date.parse(left.run.createdAt ?? "") - Date.parse(right.run.createdAt ?? ""),
  )[0];
}
