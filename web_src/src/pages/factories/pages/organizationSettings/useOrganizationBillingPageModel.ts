import { useCallback } from "react";
import { useSearchParams } from "react-router";

import type {
  OrganizationsDescribeOrganizationBillingResponse,
  OrganizationsDescribeOrganizationWorkspaceUsageResponse,
  OrganizationsHostedCreditInvoice,
  OrganizationsHostedCreditProduct,
  OrganizationsOrganizationCreditGrant,
} from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { useHostedCreditActions, useHostedCreditOwnerContactMessage } from "@/hooks/useHostedCreditActions";
import { useHostedCreditReturnRefresh } from "@/hooks/useHostedCreditReturnRefresh";
import { syncOrganizationBilling, useOrganizationBilling } from "@/hooks/useOrganizationBilling";
import { useOrganizationBillingSync } from "@/hooks/useOrganizationBillingSync";
import { useOrganizationCreditGrants } from "@/hooks/useOrganizationCreditGrants";
import { useOrganization } from "@/hooks/useOrganizationData";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import type { HostedCreditRefreshStatus } from "@/lib/hostedCredit";
import { parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";

type HostedCreditBillingActions = {
  products: OrganizationsHostedCreditProduct[];
  productsLoading: boolean;
  checkoutPending: boolean;
  businessCheckoutPending: boolean;
  portalPending: boolean;
  startCheckout: (productId: string) => Promise<void>;
  startBusinessCheckout: () => Promise<void>;
  openInvoices: () => Promise<void>;
};

export type OrganizationBillingPageModel = {
  organizationName: string;
  canManageBilling: boolean;
  billingEnabled: boolean;
  subscriptionCheckoutEnabled: boolean;
  creditPurchaseAllowed: boolean;
  plan?: string;
  trialEndsAt?: string;
  billing: HostedCreditBillingActions;
  creditRefreshStatus: HostedCreditRefreshStatus;
  billingContactMessage?: string;
  isLoading: boolean;
  error: unknown;
  billed: number;
  purchased: number;
  remaining: number;
  remainingCreditWarning: boolean;
  grantTotal: number;
  includedRemaining: number;
  purchasedRemaining: number;
  welcomeRemaining: number;
  currentPeriodEnd?: string;
  superplaneGrant: number;
  welcomeCreditExpiresAt?: string;
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
    welcomeCreditExpiresAt: spend?.welcomeCreditExpiresAt,
    hasBillingCustomer: spend?.hasBillingCustomer === true,
    invoices: spend?.invoices ?? [],
    grantTotalCents: parseWorkOrderMetric(spend?.grantTotalCents),
    spendBillingEnabled: spend?.billingEnabled === true,
  };
}

function billingFlags(billing: OrganizationsDescribeOrganizationBillingResponse | undefined) {
  return {
    plan: billing?.plan,
    trialEndsAt: billing?.trialEndsAt,
    currentPeriodEnd: billing?.currentPeriodEnd,
    includedRemaining: parseWorkOrderMetric(billing?.includedRemainingCents),
    purchasedRemaining: parseWorkOrderMetric(billing?.purchasedRemainingCents),
    welcomeRemaining: parseWorkOrderMetric(billing?.welcomeRemainingCents),
    subscriptionCheckoutEnabled: billing?.subscriptionCheckoutEnabled === true,
    creditPurchaseAllowed: billing?.creditPurchaseAllowed === true,
    describeBillingEnabled: billing?.billingEnabled === true,
  };
}

export function useOrganizationBillingPageModel(organizationId: string): OrganizationBillingPageModel {
  const [searchParams] = useSearchParams();
  const creditAdded = searchParams.get("credit") === "added";
  const subscribed = searchParams.get("subscribed") === "1";
  const { canAct } = usePermissions();
  const canManageBilling = canAct("org", "update");
  const { data: organization } = useOrganization(organizationId);
  const spend = useOrganizationWorkspaceUsage(organizationId);
  const orgBilling = useOrganizationBilling(organizationId);
  const grantsQuery = useOrganizationCreditGrants(organizationId);
  const metrics = creditMetricsFromSpend(spend.data);
  const flags = billingFlags(orgBilling.data);
  const billing = useHostedCreditActions(
    organizationId,
    canManageBilling && Boolean(organizationId) && flags.creditPurchaseAllowed,
    metrics.grantTotalCents,
    flags.creditPurchaseAllowed,
  );
  const billingEnabled = flags.describeBillingEnabled || metrics.spendBillingEnabled || billing.products.length > 0;
  const creditRefreshStatus = useHostedCreditReturnRefresh({
    organizationId,
    creditAdded,
    grantTotalCents: metrics.grantTotalCents,
    refetch: async () => {
      await Promise.all([spend.refetch(), grantsQuery.refetch(), orgBilling.refetch()]);
    },
  });
  const billingContactMessage = useHostedCreditOwnerContactMessage(organizationId, billingEnabled && !canManageBilling);

  const refetchSpend = spend.refetch;
  const refetchGrants = grantsQuery.refetch;
  const refetchBilling = orgBilling.refetch;

  const refetchBillingState = useCallback(async () => {
    await Promise.all([refetchSpend(), refetchGrants(), refetchBilling()]);
  }, [refetchSpend, refetchGrants, refetchBilling]);

  const syncFromPolar = useCallback(async () => {
    await syncOrganizationBilling(organizationId);
  }, [organizationId]);

  useOrganizationBillingSync({
    organizationId,
    subscribed,
    creditPurchaseAllowed: flags.creditPurchaseAllowed,
    sync: syncFromPolar,
    refetch: refetchBillingState,
  });

  return {
    organizationName: organization?.metadata?.name || "Organization",
    canManageBilling,
    billingEnabled,
    subscriptionCheckoutEnabled: flags.subscriptionCheckoutEnabled,
    creditPurchaseAllowed: flags.creditPurchaseAllowed,
    plan: flags.plan,
    trialEndsAt: flags.trialEndsAt,
    billing,
    creditRefreshStatus,
    billingContactMessage,
    isLoading: spend.isLoading || grantsQuery.isLoading || orgBilling.isLoading,
    error: spend.error ?? grantsQuery.error ?? orgBilling.error,
    billed: metrics.billed,
    purchased: metrics.purchased,
    remaining: metrics.remaining,
    remainingCreditWarning: metrics.remainingCreditWarning,
    grantTotal: metrics.grantTotalCents,
    includedRemaining: flags.includedRemaining,
    purchasedRemaining: flags.purchasedRemaining,
    welcomeRemaining: flags.welcomeRemaining,
    currentPeriodEnd: flags.currentPeriodEnd,
    superplaneGrant: metrics.superplaneGrant,
    welcomeCreditExpiresAt: metrics.welcomeCreditExpiresAt,
    hasBillingCustomer: metrics.hasBillingCustomer,
    invoices: metrics.invoices,
    grants: grantsQuery.data?.grants ?? [],
  };
}
