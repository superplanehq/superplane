import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { useAccount } from "@/contexts/useAccount";
import { useGooglePurchaseTracking } from "@/hooks/useGooglePurchaseTracking";
import { useSearchParams } from "react-router";

import type {
  OrganizationsDescribeOrganizationBillingResponse,
  OrganizationsDescribeOrganizationWorkspaceUsageResponse,
  OrganizationsHostedCreditInvoice,
  OrganizationsHostedCreditProduct,
  OrganizationsOrganizationCreditGrant,
  OrganizationsCreateHostedCreditCheckoutBody,
} from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { useHostedCreditActions, useHostedCreditOwnerContactMessage } from "@/hooks/useHostedCreditActions";
import { useHostedCreditReturnRefresh } from "@/hooks/useHostedCreditReturnRefresh";
import {
  useOrganizationBilling,
  useOrganizationSubscriptionActions,
  syncOrganizationBilling,
} from "@/hooks/useOrganizationBilling";
import { useOrganizationBillingSync } from "@/hooks/useOrganizationBillingSync";
import { useOrganizationCreditGrants } from "@/hooks/useOrganizationCreditGrants";
import { useOrganization } from "@/hooks/useOrganizationData";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import type { HostedCreditRefreshStatus } from "@/lib/hostedCredit";
import { billingIsPastDue, billingPaymentBlocked } from "@/pages/factories/lib/billingPlans";
import { parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";

type HostedCreditBillingActions = {
  products: OrganizationsHostedCreditProduct[];
  productsLoading: boolean;
  checkoutPending: boolean;
  portalPending: boolean;
  startCheckout: (input: OrganizationsCreateHostedCreditCheckoutBody) => Promise<void>;
  openInvoices: () => Promise<void>;
};

export type BillingSeatSummary = {
  purchased?: number;
  occupied?: number;
  available?: number;
  pendingInvitations?: number;
  invitable?: number;
  planMaximum?: number;
  purchasable?: number;
  expectedRenewalSeats?: number;
  currentMonthlyTotalCents?: string;
  expectedRenewalCents?: string;
  /** Set while a failed or paused payment stops new members. */
  paymentBlock?: BillingPaymentBlock;
};

/** Why payment stops new members: a failed renewal payment or a paused subscription. */
export type BillingPaymentBlock = "past_due" | "paused";

export type OrganizationBillingPageModel = {
  seats?: BillingSeatSummary;
  organizationName: string;
  canManageBilling: boolean;
  billingEnabled: boolean;
  subscriptionCheckoutEnabled: boolean;
  creditPurchaseAllowed: boolean;
  plan?: string;
  planCategory?: string;
  planName?: string;
  planPriceCents?: string;
  planPerSeat?: boolean;
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
  adminRemaining: number;
  currentPeriodEnd?: string;
  planSource?: string;
  cancelAtPeriodEnd: boolean;
  cancelPending: boolean;
  keepPending: boolean;
  onCancelSubscription: () => Promise<void>;
  onKeepSubscription: () => Promise<void>;
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

function billingSeats(billing: OrganizationsDescribeOrganizationBillingResponse): BillingSeatSummary | undefined {
  // The server sends stored seats for a seat-based plan while payment is blocked, even after the plan row changes.
  const paymentBlock = billingSeatPaymentBlock(billing);
  if (!billing.billingEnabled || !(billing.planPerSeat || paymentBlock)) {
    return undefined;
  }
  return {
    purchased: billing.purchasedSeats,
    occupied: billing.occupiedSeats,
    available: billing.availableSeats,
    pendingInvitations: billing.pendingInvitationSeats,
    invitable: billing.invitableSeats,
    planMaximum: billing.planMaximumSeats,
    purchasable: billing.purchasableSeats,
    expectedRenewalSeats: billing.expectedRenewalSeats,
    currentMonthlyTotalCents: billing.currentMonthlyTotalCents,
    expectedRenewalCents: billing.expectedRenewalCents,
    paymentBlock,
  };
}

function billingSeatPaymentBlock(
  billing: OrganizationsDescribeOrganizationBillingResponse,
): BillingPaymentBlock | undefined {
  if (billing.purchasedSeats === undefined) {
    return undefined;
  }
  if (billingIsPastDue(billing.polarSubscriptionStatus)) {
    return "past_due";
  }
  return billingPaymentBlocked(billing.polarSubscriptionStatus) ? "paused" : undefined;
}

function billingFlags(billing: OrganizationsDescribeOrganizationBillingResponse = {}) {
  return {
    plan: billing.plan,
    planCategory: billing.planCategory,
    planName: billing.planName,
    planPriceCents: billing.planPriceCents,
    planPerSeat: billing.planPerSeat,
    seats: billingSeats(billing),
    trialEndsAt: billing.trialEndsAt,
    currentPeriodEnd: billing.currentPeriodEnd,
    includedRemaining: parseWorkOrderMetric(billing.includedRemainingCents),
    purchasedRemaining: parseWorkOrderMetric(billing.purchasedRemainingCents),
    welcomeRemaining: parseWorkOrderMetric(billing.welcomeRemainingCents),
    adminRemaining: parseWorkOrderMetric(billing.adminRemainingCents),
    subscriptionCheckoutEnabled: billing.subscriptionCheckoutEnabled === true,
    creditPurchaseAllowed: billing.creditPurchaseAllowed === true,
    describeBillingEnabled: billing.billingEnabled === true,
    planSource: billing.planSource,
    cancelAtPeriodEnd: billing.cancelAtPeriodEnd === true,
  };
}

export function useOrganizationBillingPageModel(organizationId: string): OrganizationBillingPageModel {
  const [searchParams] = useSearchParams();
  const creditAdded = searchParams.get("credit") === "added";
  const subscribed = searchParams.get("subscribed") === "1";
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canManageBilling = canAct("org", "update");
  const { data: organization } = useOrganization(organizationId);
  const spend = useOrganizationWorkspaceUsage(organizationId);
  const orgBilling = useOrganizationBilling(organizationId);
  const grantsQuery = useOrganizationCreditGrants(organizationId);
  const { account } = useAccount();
  const metrics = creditMetricsFromSpend(spend.data);
  const purchasePending = useGooglePurchaseTracking({
    checkoutID: searchParams.get("checkout_id") ?? "",
    invoices: metrics.invoices,
    enabled: (creditAdded || subscribed) && canManageBilling && !account?.support_session?.active,
  });
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
    pollingEnabled: false,
  });
  const billingContactMessage = useHostedCreditOwnerContactMessage(organizationId, billingEnabled && !canManageBilling);

  const refetchSpend = spend.refetch;
  const refetchGrants = grantsQuery.refetch;
  const refetchBilling = orgBilling.refetch;

  const refetchBillingState = useCallback(async () => {
    await Promise.all([refetchSpend(), refetchGrants(), refetchBilling()]);
  }, [refetchSpend, refetchGrants, refetchBilling]);

  const syncFromPolar = useBillingSyncCallback(organizationId);

  const subscription = useOrganizationSubscriptionActions(organizationId);

  useOrganizationBillingSync({
    organizationId,
    subscribed,
    creditPurchaseAllowed: flags.creditPurchaseAllowed,
    returnRefreshPending: purchasePending || creditRefreshStatus === "refreshing",
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
    planCategory: flags.planCategory,
    planName: flags.planName,
    planPriceCents: flags.planPriceCents,
    planPerSeat: flags.planPerSeat,
    seats: flags.seats,
    trialEndsAt: flags.trialEndsAt,
    billing,
    creditRefreshStatus,
    billingContactMessage,
    isLoading: permissionsLoading || spend.isLoading || grantsQuery.isLoading || orgBilling.isLoading,
    error: spend.error ?? grantsQuery.error ?? orgBilling.error,
    billed: metrics.billed,
    purchased: metrics.purchased,
    remaining: metrics.remaining,
    remainingCreditWarning: metrics.remainingCreditWarning,
    grantTotal: metrics.grantTotalCents,
    includedRemaining: flags.includedRemaining,
    purchasedRemaining: flags.purchasedRemaining,
    welcomeRemaining: flags.welcomeRemaining,
    adminRemaining: flags.adminRemaining,
    currentPeriodEnd: flags.currentPeriodEnd,
    planSource: flags.planSource,
    cancelAtPeriodEnd: flags.cancelAtPeriodEnd,
    cancelPending: subscription.cancelPending,
    keepPending: subscription.keepPending,
    onCancelSubscription: subscription.onCancelSubscription,
    onKeepSubscription: subscription.onKeepSubscription,
    superplaneGrant: metrics.superplaneGrant,
    welcomeCreditExpiresAt: metrics.welcomeCreditExpiresAt,
    hasBillingCustomer: metrics.hasBillingCustomer,
    invoices: metrics.invoices,
    grants: grantsQuery.data?.grants ?? [],
  };
}

function useBillingSyncCallback(organizationId: string) {
  const queryClient = useQueryClient();
  return useCallback(() => syncOrganizationBilling(organizationId, queryClient), [organizationId, queryClient]);
}
