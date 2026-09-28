import type { FactoriesWorkOrderExecution } from "@/api-client";
import { Link } from "@/components/Link/link";
import { cn } from "@/lib/utils";

import { factorySettingsSectionPath } from "./lib/factoryPagePaths";
import { workOrderExecutionCreditFailure } from "./lib/workOrderFailureReason";

interface WorkOrderCreditFailureNoticeProps {
  organizationId: string;
  factoryKey: string;
  execution: Pick<FactoriesWorkOrderExecution, "result" | "failureReason">;
  className?: string;
}

/**
 * Tells the user that SuperPlane hosted credit stopped a step and links to
 * billing. Renders nothing for other results and failures.
 */
export function WorkOrderCreditFailureNotice({
  organizationId,
  factoryKey,
  execution,
  className,
}: WorkOrderCreditFailureNoticeProps) {
  const failure = workOrderExecutionCreditFailure(execution);
  if (!failure) {
    return null;
  }

  const billingHref = factorySettingsSectionPath(organizationId, factoryKey, "organization", "billing");

  return (
    <p
      data-testid="work-order-credit-failure"
      className={cn("text-[12px] leading-relaxed text-[color:var(--status-danger)]", className)}
    >
      {failure.message}{" "}
      <Link href={billingHref} className="font-medium underline underline-offset-2 hover:no-underline">
        {failure.actionLabel}
      </Link>
    </p>
  );
}
