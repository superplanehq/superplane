import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { OPEN_WORK_ORDER_CHECKS } from "../../__fixtures__/workOrderCheckFixtures";
import { REVIEW_CANDIDATE_WORK_ORDERS } from "../onboarding/first-run/reviewCandidates";
import { presentWorkOrderChecks } from "../../lib/workOrderChecks";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";

const COMPARE_CLIENT = new QueryClient({
  defaultOptions: { queries: { retry: false, refetchOnWindowFocus: false } },
});

/**
 * Storybook preview of the working two-column work-order tab.
 * Description on the left. Checks and artifacts on the right. No footer.
 */
export function WorkOrderPopupTabsCompare() {
  const fixture = {
    ...splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0], { checks: OPEN_WORK_ORDER_CHECKS }),
    checks: presentWorkOrderChecks(
      OPEN_WORK_ORDER_CHECKS.filter((check) => check.key === "risk-review" || check.key === "code-coverage"),
    ),
  };

  return (
    <QueryClientProvider client={COMPARE_CLIENT}>
      <WorkOrderSplitRunPopup fixture={fixture} />
    </QueryClientProvider>
  );
}
