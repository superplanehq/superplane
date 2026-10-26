import type {
  FactoriesDescribeFactoryUsageResponse,
  OrganizationsDescribeOrganizationBillingResponse,
} from "@/api-client";
import { useBillingProductEnabled } from "@/hooks/useBillingProductEnabled";
import { useFactoryLineRunnerModels } from "@/hooks/useFactoryLineRunnerModels";
import { useFactoryUsage } from "@/hooks/useFactoryUsage";
import { useOrganizationBilling } from "@/hooks/useOrganizationBilling";

import { billingIsPastDue } from "../../lib/billingPlans";
import { workOrderCreditFailureCopy, type WorkOrderCreditFailureReason } from "../../lib/workOrderFailureReason";
import type { SplitRunFooter, SplitRunFooterNote } from "./splitRunFooter";

/** Keep Start and its explanation in sync with the current billing state. */
export function useWorkOrderStartFooter(
  footer: SplitRunFooter,
  organizationId?: string,
  factoryId?: string,
  lineName?: string,
) {
  const billingEnabled = useBillingProductEnabled();
  const checkStart = billingEnabled && footer.actions.some((action) => action.kind === "start");
  const billing = useOrganizationBilling(checkStart ? organizationId : undefined);
  const usage = useFactoryUsage(checkStart ? (organizationId ?? "") : "", checkStart ? (factoryId ?? "") : "");
  const runnerModels = useFactoryLineRunnerModels(organizationId, factoryId, lineName, checkStart);
  const block = checkStart ? runStartBlock(billing.data, usage.data, runnerModels.data?.usesHostedFunding) : undefined;
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
  usesHostedFunding: boolean | undefined,
): SplitRunFooterNote | undefined {
  if (billing?.hostedExecutionAllowed === false) {
    if (billing.polarSubscriptionStatus === "paused") {
      return creditStartBlock("subscription_paused");
    }
    if (billing.requirePaymentMethod && !billing.hasPaymentMethod) {
      return {
        headline: "Payment method required",
        text: "Add a payment method in Billing to start this task.",
        cta: { label: "View billing", destination: "billing" },
      };
    }
    if (billingIsPastDue(billing.polarSubscriptionStatus)) {
      return creditStartBlock("payment_past_due");
    }
    return creditStartBlock("hosted_subscription_required");
  }

  // A line whose runners all run on the organization's own provider keys
  // never spends hosted credit, so the hosted credit and workspace budget
  // limits do not block its Start. An unknown funding state keeps the hosted
  // checks.
  if (usesHostedFunding === false) {
    return undefined;
  }

  // The server gates runs on exact micros, so use its availability flags
  // instead of the rounded whole-cent balances.
  const creditAvailable = billing?.hostedCreditAvailable ?? usage?.hostedCreditAvailable;
  if (creditAvailable === false) {
    return creditStartBlock("no_hosted_credit");
  }
  if (usage?.hostedSpendBudgetCents != null && usage.factoryHostedBudgetAvailable === false) {
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
