import { boardConfidenceFixture } from "./boardConfidenceFixture";
import type { FactoriesFixture } from "./factoryPageResponses";
import { LINE_BOARD_VERIFY_ENUM_ORDER, LINE_BOARD_VERIFY_PR_REVIEW_ORDER } from "./lineMetricsFactoriesFixture";

/** Full app task view and Board share the same check and pull request responses. */
export function prReadyFactoriesFixture(assessment: "ready" | "review" | "partial"): FactoriesFixture {
  const orderId = LINE_BOARD_VERIFY_PR_REVIEW_ORDER.id!;
  const sourceId = assessment === "ready" ? orderId : LINE_BOARD_VERIFY_ENUM_ORDER.id!;
  const checks = (boardConfidenceFixture.checksByOrderId?.[sourceId] ?? []).map((check) => ({
    ...check,
    id: `${orderId}-${check.key}`,
  }));
  return {
    ...boardConfidenceFixture,
    checksByOrderId: {
      ...boardConfidenceFixture.checksByOrderId,
      [orderId]: assessment === "partial" ? checks.slice(0, 2) : checks,
    },
  };
}
