import type { FactoriesWorkOrderCheck } from "@/api-client";

import type { PlanningSessionPayload } from "../pages/planningSessionView";
import { mergeConfidenceCheckLevel } from "../lib/mergeConfidenceScore";
import { planningReviewLevel } from "../lib/planningReviewScore";
import { workOrderCheckStatus, type WorkOrderCheckLevel } from "../lib/workOrderChecks";
import {
  DRAFT_WORK_ORDER,
  RUNNING_WORK_ORDER,
  PRIMARY_FACTORY_ID,
  type FactoriesFixture,
} from "./factoryPageResponses";
import {
  lineMetricsFactoriesFixture,
  LINE_BOARD_VERIFY_ENUM_ORDER,
  LINE_BOARD_VERIFY_PR_REVIEW_ORDER,
} from "./lineMetricsFactoriesFixture";

const CHECK_LEVEL: Record<WorkOrderCheckLevel, FactoriesWorkOrderCheck["level"]> = {
  positive: "LEVEL_POSITIVE",
  neutral: "LEVEL_NEUTRAL",
  caution: "LEVEL_CAUTION",
  critical: "LEVEL_CRITICAL",
};

function boardCheck(
  orderId: string,
  check: { key: string; name: string; score: number; maxScore: number; level: WorkOrderCheckLevel },
): FactoriesWorkOrderCheck {
  return {
    ...check,
    id: `${orderId}-${check.key}`,
    level: CHECK_LEVEL[check.level],
    summary: `${check.name}: ${workOrderCheckStatus(check).label}.`,
    updatedAt: new Date(Date.now() - 120_000).toISOString(),
  };
}

function planningChecks(orderId: string, scores: [number, number, number]): FactoriesWorkOrderCheck[] {
  return [
    { key: "clarity", name: "Clarity", score: scores[0] },
    { key: "complexity", name: "Complexity", score: scores[1] },
    { key: "verifiability", name: "Verifiability", score: scores[2] },
  ].map((check) => boardCheck(orderId, { ...check, maxScore: 3, level: planningReviewLevel(check.score) }));
}

function mergeChecks(orderId: string, scores: [number, number, number, number, number]): FactoriesWorkOrderCheck[] {
  return [
    { key: "risk-review", name: "Blast radius", score: scores[0] },
    { key: "performance-review", name: "Performance", score: scores[1] },
    { key: "security-review", name: "Security", score: scores[2] },
    { key: "drift-review", name: "Plan drift", score: scores[3] },
    { key: "reversibility-review", name: "Reversibility", score: scores[4] },
  ].map((check) =>
    boardCheck(orderId, { ...check, maxScore: 5, level: mergeConfidenceCheckLevel(check.key, check.score) }),
  );
}

const planningChecksByOrderId = {
  "wo-review-pay-842": planningChecks("wo-review-pay-842", [3, 3, 3]),
  "wo-review-pay-844": planningChecks("wo-review-pay-844", [3, 2, 3]),
  "wo-review-pay-845": planningChecks("wo-review-pay-845", [1, 3, 2]),
};

/** Shared list/detail responses: ready, caution, critical, and unscored checks. */
export const boardConfidenceFixture: FactoriesFixture = {
  ...lineMetricsFactoriesFixture,
  planningSessionsByWorkOrderId: Object.fromEntries(
    Object.entries(planningChecksByOrderId).map(([orderId, checks]): [string, PlanningSessionPayload] => {
      const order = lineMetricsFactoriesFixture.workOrdersByFactoryId[PRIMARY_FACTORY_ID].find(
        (entry) => entry.id === orderId,
      );
      return [
        orderId,
        {
          id: `session-${orderId}`,
          factoryId: PRIMARY_FACTORY_ID,
          state: "ended",
          kind: "PLANNING_SESSION_KIND_WORK_ORDER_ANALYSIS",
          draft: { title: order?.title, description: order?.description, workOrderId: orderId },
          messages: [
            {
              id: `plan-${orderId}`,
              role: "plan",
              text: JSON.stringify({ score: Math.min(...checks.map((check) => check.score ?? 0)) }),
              createdAt: checks[0].updatedAt,
            },
            {
              id: `summary-${orderId}`,
              role: "agent",
              text: "I completed the planning checks. Review the results before implementation.",
              createdAt: checks[0].updatedAt,
            },
          ],
        },
      ];
    }),
  ),
  checksByOrderId: {
    ...lineMetricsFactoriesFixture.checksByOrderId,
    ...planningChecksByOrderId,
    [DRAFT_WORK_ORDER.id!]: [],
    [RUNNING_WORK_ORDER.id!]: planningChecks(RUNNING_WORK_ORDER.id!, [3, 3, 3]),
    [LINE_BOARD_VERIFY_ENUM_ORDER.id!]: mergeChecks(LINE_BOARD_VERIFY_ENUM_ORDER.id!, [1, 3, 2, 4, 5]),
    [LINE_BOARD_VERIFY_PR_REVIEW_ORDER.id!]: mergeChecks(LINE_BOARD_VERIFY_PR_REVIEW_ORDER.id!, [1, 5, 5, 1, 5]),
  },
};
