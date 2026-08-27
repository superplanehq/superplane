import type { FactoriesFactoryPrFeedbackHandler, FactoriesFactoryPullRequest } from "@/api-client";
import { useMemo } from "react";

import { activePRFeedbackWorkOrderIds, type PRFeedbackLogRun } from "./prFeedbackSettingsModel";
import { prFeedbackRunTitle } from "../lib/workOrderPullRequest";

export function useActivePRFeedbackWorkOrderIds(pullRequests: FactoriesFactoryPullRequest[]): ReadonlySet<string> {
  return useMemo(() => activePRFeedbackWorkOrderIds(pullRequests), [pullRequests]);
}

export function useWorkOrderPRFeedbackLog(
  pullRequests: FactoriesFactoryPullRequest[],
  handlers: FactoriesFactoryPrFeedbackHandler[] = [],
): PRFeedbackLogRun[] {
  return useMemo(() => prFeedbackLogRunsFromPullRequests(pullRequests, handlers), [handlers, pullRequests]);
}

export function prFeedbackLogRunsFromPullRequests(
  pullRequests: FactoriesFactoryPullRequest[],
  handlers: FactoriesFactoryPrFeedbackHandler[] = [],
): PRFeedbackLogRun[] {
  const handlerNameByCanvasId = new Map(
    handlers.flatMap((handler) =>
      handler.canvasId ? [[handler.canvasId, handler.name?.trim() || undefined] as const] : [],
    ),
  );

  return pullRequests.flatMap((pullRequest) =>
    [...(pullRequest.runs ?? [])]
      .filter((run) => Boolean(run.id && run.canvasId))
      .sort((left, right) => Date.parse(left.createdAt ?? "") - Date.parse(right.createdAt ?? ""))
      .map((run) => ({
        canvasId: run.canvasId ?? "",
        handlerName: handlerNameByCanvasId.get(run.canvasId ?? ""),
        pullRequestNumber: pullRequest.number,
        run,
      })),
  );
}

export { prFeedbackRunTitle };
