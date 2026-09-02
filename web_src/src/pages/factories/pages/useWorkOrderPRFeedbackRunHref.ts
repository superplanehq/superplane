import type { FactoriesFactoryPrFeedbackHandler, FactoriesFactoryPullRequest } from "@/api-client";
import { useMemo } from "react";

import { firstPositiveWorkOrderMetric } from "../lib/workOrderUsage";
import {
  addressingFeedbackLabelsByWorkOrder,
  checksPassedWorkOrderIds,
  fixesPausedWorkOrderIds,
  prFeedbackActivityAttemptLabel,
  prFeedbackActivityKind,
  prFeedbackActivityLabel,
  waitingOnChecksWorkOrderIds,
  type PRFeedbackLogRun,
} from "./prFeedbackSettingsModel";
import { prFeedbackRunTitle } from "../lib/workOrderPullRequest";

export function usePRFeedbackWorkOrderAttention(pullRequests: FactoriesFactoryPullRequest[]): {
  addressingFeedbackOrderIds: ReadonlySet<string>;
  addressingFeedbackLabels: ReadonlyMap<string, string>;
  waitingOnChecksOrderIds: ReadonlySet<string>;
  checksPassedOrderIds: ReadonlySet<string>;
  fixesPausedOrderIds: ReadonlySet<string>;
} {
  return useMemo(() => {
    const addressingFeedbackLabels = addressingFeedbackLabelsByWorkOrder(pullRequests);
    return {
      addressingFeedbackOrderIds: new Set(addressingFeedbackLabels.keys()),
      addressingFeedbackLabels,
      waitingOnChecksOrderIds: waitingOnChecksWorkOrderIds(pullRequests),
      checksPassedOrderIds: checksPassedWorkOrderIds(pullRequests),
      fixesPausedOrderIds: fixesPausedWorkOrderIds(pullRequests),
    };
  }, [pullRequests]);
}

export function useActivePRFeedbackWorkOrderIds(pullRequests: FactoriesFactoryPullRequest[]): ReadonlySet<string> {
  return usePRFeedbackWorkOrderAttention(pullRequests).addressingFeedbackOrderIds;
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

  return pullRequests.flatMap((pullRequest) => {
    const usageByRunId = new Map(
      (pullRequest.runs ?? []).flatMap((linked) =>
        linked.run?.id
          ? [[linked.run.id, { costCents: linked.costCents, totalTokens: linked.totalTokens }] as const]
          : [],
      ),
    );
    const activities = pullRequest.activities ?? [];
    const entries =
      activities.length > 0
        ? activities.flatMap((activity) => {
            const run = activity.run;
            if (!run?.id || !run.canvasId) {
              return [];
            }
            const usage = usageByRunId.get(run.id);
            return [
              {
                canvasId: run.canvasId,
                handlerName: handlerNameByCanvasId.get(run.canvasId),
                pullRequestNumber: pullRequest.number,
                description: prFeedbackActivityLabel(activity),
                attemptLabel: prFeedbackActivityAttemptLabel(activity),
                costCents: firstPositiveWorkOrderMetric(activity.costCents, usage?.costCents),
                totalTokens: firstPositiveWorkOrderMetric(activity.totalTokens, usage?.totalTokens),
                kind: prFeedbackActivityKind(activity),
                run,
              },
            ];
          })
        : (pullRequest.runs ?? []).flatMap((linked) => {
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
          });

    return entries.sort((left, right) => Date.parse(left.run.createdAt ?? "") - Date.parse(right.run.createdAt ?? ""));
  });
}

export { prFeedbackRunTitle };
