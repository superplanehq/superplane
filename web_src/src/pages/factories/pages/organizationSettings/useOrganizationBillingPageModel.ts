import { useSearchParams } from "react-router";

import type {
  OrganizationsDescribeOrganizationWorkspaceUsageResponse,
  OrganizationsHostedCreditInvoice,
  OrganizationsHostedCreditProduct,
  OrganizationsOrganizationCreditGrant,
} from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { useHostedCreditActions, useHostedCreditOwnerContactMessage } from "@/hooks/useHostedCreditActions";
import { useHostedCreditReturnRefresh } from "@/hooks/useHostedCreditReturnRefresh";
import { useOrganizationCreditGrants } from "@/hooks/useOrganizationCreditGrants";
import { useOrganization } from "@/hooks/useOrganizationData";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import type { HostedCreditRefreshStatus } from "@/lib/hostedCredit";
import { parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";

type HostedCreditBillingActions = {
  products: OrganizationsHostedCreditProduct[];
  productsLoading: boolean;
  checkoutPending: boolean;
  portalPending: boolean;
  startCheckout: (productId: string) => Promise<void>;
  openInvoices: () => Promise<void>;
};

export type OrganizationBillingPageModel = {
  organizationName: string;
  canManageBilling: boolean;
  billingEnabled: boolean;
  billing: HostedCreditBillingActions;
  creditRefreshStatus: HostedCreditRefreshStatus;
  billingContactMessage?: string;
  isLoading: boolean;
  error: unknown;
  billed: number;
  purchased: number;
  remaining: number;
  remainingCreditWarning: boolean;
  superplaneGrant: number;
  hasBillingCustomer: boolean;
  invoices: OrganizationsHostedCreditInvoice[];
  grants: OrganizationsOrganizationCreditGrant[];
};

function creditMetricsFromSpend(spend: OrganizationsDescribeOrganizationWorkspaceUsageResponse | undefined) {
  return {
    billed: parseWorkOrderMetric(spend?.hostedBilledCents),
    purchased: parseWorkOrderMetric(spend?.purchasedCreditCents),
    remaining: parseWorkOrderMetric(spend?.remainingCreditCents),
    remainingCreditWarning: spend?.remainingCreditWarning === true,
    superplaneGrant: parseWorkOrderMetric(spend?.superplaneGrantCents),
    hasBillingCustomer: spend?.hasBillingCustomer === true,
    invoices: spend?.invoices ?? [],
    grantTotalCents: parseWorkOrderMetric(spend?.grantTotalCents),
    spendBillingEnabled: spend?.billingEnabled === true,
  };
}

export function useOrganizationBillingPageModel(organizationId: string): OrganizationBillingPageModel {
  const [searchParams] = useSearchParams();
  const creditAdded = searchParams.get("credit") === "added";
  const { canAct } = usePermissions();
  const canManageBilling = canAct("org", "update");
  const { data: organization } = useOrganization(organizationId);
  const spend = useOrganizationWorkspaceUsage(organizationId);
  const grantsQuery = useOrganizationCreditGrants(organizationId);
  const metrics = creditMetricsFromSpend(spend.data);
  // Owners fetch packs immediately. Do not wait for the spend report flag.
  const billing = useHostedCreditActions(
    organizationId,
    canManageBilling && Boolean(organizationId),
    metrics.grantTotalCents,
  );
  const billingEnabled = metrics.spendBillingEnabled || billing.products.length > 0;
  const creditRefreshStatus = useHostedCreditReturnRefresh({
    organizationId,
    creditAdded,
    grantTotalCents: metrics.grantTotalCents,
    refetch: async () => {
      await Promise.all([spend.refetch(), grantsQuery.refetch()]);
    },
  });
  const billingContactMessage = useHostedCreditOwnerContactMessage(organizationId, billingEnabled && !canManageBilling);

  return {
    organizationName: organization?.metadata?.name || "Organization",
    canManageBilling,
    billingEnabled,
    billing,
    creditRefreshStatus,
    billingContactMessage,
    isLoading: spend.isLoading || grantsQuery.isLoading,
    error: spend.error ?? grantsQuery.error,
    billed: metrics.billed,
    purchased: metrics.purchased,
    remaining: metrics.remaining,
    remainingCreditWarning: metrics.remainingCreditWarning,
    superplaneGrant: metrics.superplaneGrant,
    hasBillingCustomer: metrics.hasBillingCustomer,
    invoices: metrics.invoices,
    grants: grantsQuery.data?.grants ?? [],
  };
}
