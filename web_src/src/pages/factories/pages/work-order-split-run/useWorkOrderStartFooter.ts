import type {
  FactoriesDescribeFactoryUsageResponse,
  OrganizationsDescribeOrganizationBillingResponse,
} from "@/api-client";
import { useBillingProductEnabled } from "@/hooks/useBillingProductEnabled";
import { useFactoryUsage } from "@/hooks/useFactoryUsage";
import { useOrganizationBilling } from "@/hooks/useOrganizationBilling";

import { workOrderCreditFailureCopy, type WorkOrderCreditFailureReason } from "../../lib/workOrderFailureReason";
import { parseWorkOrderMetric } from "../../lib/workOrderUsage";
import type { SplitRunFooter, SplitRunFooterNote } from "./splitRunFooter";

/** Keep Start and its explanation in sync with the current billing state. */
export function useWorkOrderStartFooter(footer: SplitRunFooter, organizationId?: string, factoryId?: string) {
  const billingEnabled = useBillingProductEnabled();
  const checkStart = billingEnabled && footer.actions.some((action) => action.kind === "start");
  const billing = useOrganizationBilling(checkStart ? organizationId : undefined);
  const usage = useFactoryUsage(checkStart ? (organizationId ?? "") : "", checkStart ? (factoryId ?? "") : "");
  const block = checkStart ? runStartBlock(billing.data, usage.data) : undefined;
  if (!block) {
    return footer;
  }
  return {
    ...footer,
    note: block,
    actions: footer.actions.map((action) =>
      action.kind === "start" ? { ...action, disabled: true, tooltip: block.text } : action,
    ),
  };
}

function runStartBlock(
  billing: OrganizationsDescribeOrganizationBillingResponse | undefined,
  usage: FactoriesDescribeFactoryUsageResponse | undefined,
): SplitRunFooterNote | undefined {
  if (billing?.hostedExecutionAllowed === false) {
    if (billing.requirePaymentMethod && !billing.hasPaymentMethod) {
      return {
        headline: "Payment method required",
        text: "Add a payment method in Billing to start this task.",
        cta: { label: "View billing", destination: "billing" },
      };
    }
    return creditStartBlock("hosted_subscription_required");
  }

  const remainingCredit = billing?.remainingCreditCents ?? usage?.remainingCreditCents;
  if (remainingCredit != null && parseWorkOrderMetric(remainingCredit) <= 0) {
    return creditStartBlock("no_hosted_credit");
  }
  if (
    usage?.hostedSpendBudgetCents != null &&
    usage.factoryRemainingCreditCents != null &&
    parseWorkOrderMetric(usage.factoryRemainingCreditCents) <= 0
  ) {
    return creditStartBlock("workspace_budget_empty");
  }
  return undefined;
}

function creditStartBlock(reason: WorkOrderCreditFailureReason): SplitRunFooterNote | undefined {
  const failure = workOrderCreditFailureCopy(reason);
  if (!failure) {
    return undefined;
  }
  return {
    headline: failure.label,
    text: failure.message,
    cta: { label: failure.actionLabel, destination: "billing" },
  };
}
