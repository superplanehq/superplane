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
      .flatMap((linked) => {
        const run = linked.run;
        if (!run?.id || !run.canvasId) {
          return [];
        }
        return [
          {
            canvasId: run.canvasId,
            handlerName: handlerNameByCanvasId.get(run.canvasId),
            pullRequestNumber: pullRequest.number,
            description: linked.description,
            costCents: linked.costCents,
            totalTokens: linked.totalTokens,
            run,
          },
        ];
      })
      .sort((left, right) => Date.parse(left.run.createdAt ?? "") - Date.parse(right.run.createdAt ?? "")),
  );
}

export { prFeedbackRunTitle };
