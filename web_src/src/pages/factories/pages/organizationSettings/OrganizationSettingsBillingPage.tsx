import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_NEW_BILLING } from "@/lib/experimentalFeatures";
import { type ReactNode } from "react";
import { Navigate, useParams } from "react-router";

import { useBillingProductEnabled } from "@/hooks/useBillingProductEnabled";

import type {
  OrganizationsCreateHostedCreditCheckoutBody,
  OrganizationsHostedCreditInvoice,
  OrganizationsHostedCreditProduct,
  OrganizationsOrganizationCreditGrant,
} from "@/api-client";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { type HostedCreditRefreshStatus } from "@/lib/hostedCredit";

import { factorySettingsSectionPath } from "../../lib/factoryPagePaths";
import { FactorySettingsCard, FactorySettingsPageFrame } from "../settings/FactorySettingsCard";
import { BillingHistory } from "./BillingHistory";
import { BillingSeatPurchaseFlow } from "./BillingSeatPurchaseFlow";
import { BillingPaymentSettings } from "./BillingPaymentSettings";
import { BillingPlansSection } from "./BillingPlansSection";
import { HostedCreditRemainingCard } from "./HostedCreditRemainingCard";
import { useOrganizationBillingPageModel } from "./useOrganizationBillingPageModel";

export function OrganizationSettingsBillingPage() {
  const { organizationId = "", factoryKey = "" } = useParams<{ organizationId: string; factoryKey: string }>();
  const billingProductEnabled = useBillingProductEnabled();
  const newBilling = useExperimentalFeature(organizationId).has(FEATURE_NEW_BILLING);
  const model = useOrganizationBillingPageModel(organizationId);
  const packs = model.billing.products;

  usePageTitle(["Billing", model.organizationName]);
  useReportPageReady(!model.isLoading, { failed: Boolean(model.error) });

  if (!billingProductEnabled) {
    return <Navigate to={factorySettingsSectionPath(organizationId, factoryKey, "organization", "general")} replace />;
  }

  return (
    <FactorySettingsPageFrame title="Billing">
      <BillingSeatPurchaseFlow
        organizationId={organizationId}
        newBilling={newBilling}
        canManageBilling={model.canManageBilling}
        seats={model.seats}
        currentPeriodEnd={model.currentPeriodEnd}
        cancelAtPeriodEnd={model.cancelAtPeriodEnd}
        loading={model.isLoading}
      >
        {(seatsCard, dialog) => (
          <>
            <BillingPageBody
              organizationId={organizationId}
              newBilling={newBilling}
              billingContactMessage={model.billingContactMessage}
              billingEnabled={model.billingEnabled}
              businessCheckoutPending={model.billing.businessCheckoutPending}
              canManageBilling={model.canManageBilling}
              checkoutPending={model.billing.checkoutPending}
              creditPurchaseAllowed={model.creditPurchaseAllowed}
              creditRefreshStatus={model.creditRefreshStatus}
              error={model.error}
              grants={model.grants}
              hasBillingCustomer={model.hasBillingCustomer}
              invoices={model.invoices}
              isLoading={model.isLoading}
              packs={packs}
              plan={model.plan}
              planCategory={model.planCategory}
              planName={model.planName}
              planPriceCents={model.planPriceCents}
              planPerSeat={model.planPerSeat}
              seatsCard={seatsCard}
              planSource={model.planSource}
              cancelAtPeriodEnd={model.cancelAtPeriodEnd}
              subscriptionPending={model.cancelPending || model.keepPending}
              portalPending={model.billing.portalPending}
              purchased={model.purchased}
              remaining={model.remaining}
              includedRemaining={model.includedRemaining}
              purchasedRemaining={model.purchasedRemaining}
              welcomeRemaining={model.welcomeRemaining}
              adminRemaining={model.adminRemaining}
              currentPeriodEnd={model.currentPeriodEnd}
              trialEndsAt={model.trialEndsAt}
              welcomeCreditExpiresAt={model.welcomeCreditExpiresAt}
              onAddCredit={model.billing.startCheckout}
              onSubscribe={model.billing.startBusinessCheckout}
              onCancelSubscription={model.onCancelSubscription}
              onKeepSubscription={model.onKeepSubscription}
              onManageInvoices={model.billing.openInvoices}
            />
            {dialog}
          </>
        )}
      </BillingSeatPurchaseFlow>
    </FactorySettingsPageFrame>
  );
}

type BillingPageBodyProps = {
  organizationId: string;
  newBilling: boolean;
  billingContactMessage?: string;
  billingEnabled: boolean;
  businessCheckoutPending: boolean;
  canManageBilling: boolean;
  checkoutPending: boolean;
  creditPurchaseAllowed: boolean;
  creditRefreshStatus: HostedCreditRefreshStatus;
  error: unknown;
  grants: OrganizationsOrganizationCreditGrant[];
  hasBillingCustomer: boolean;
  invoices: OrganizationsHostedCreditInvoice[];
  isLoading: boolean;
  packs: OrganizationsHostedCreditProduct[];
  plan?: string;
  planCategory?: string;
  planName?: string;
  planPriceCents?: string;
  planPerSeat?: boolean;
  seatsCard?: ReactNode;
  planSource?: string;
  cancelAtPeriodEnd: boolean;
  subscriptionPending: boolean;
  portalPending: boolean;
  purchased: number;
  remaining: number;
  includedRemaining: number;
  purchasedRemaining: number;
  welcomeRemaining: number;
  adminRemaining: number;
  currentPeriodEnd?: string;
  trialEndsAt?: string;
  welcomeCreditExpiresAt?: string;
  onAddCredit: (input: OrganizationsCreateHostedCreditCheckoutBody) => Promise<void>;
  onSubscribe: () => void | Promise<void>;
  onCancelSubscription: () => void | Promise<void>;
  onKeepSubscription: () => void | Promise<void>;
  onManageInvoices: () => void | Promise<void>;
};

function BillingPageBody({
  organizationId,
  newBilling,
  billingContactMessage,
  billingEnabled,
  businessCheckoutPending,
  canManageBilling,
  checkoutPending,
  creditPurchaseAllowed,
  creditRefreshStatus,
  error,
  grants,
  hasBillingCustomer,
  invoices,
  isLoading,
  packs,
  plan,
  planCategory,
  planName,
  planPriceCents,
  planPerSeat,
  seatsCard,
  planSource,
  cancelAtPeriodEnd,
  subscriptionPending,
  portalPending,
  purchased,
  remaining,
  includedRemaining,
  purchasedRemaining,
  welcomeRemaining,
  adminRemaining,
  currentPeriodEnd,
  trialEndsAt,
  welcomeCreditExpiresAt,
  onAddCredit,
  onSubscribe,
  onCancelSubscription,
  onKeepSubscription,
  onManageInvoices,
}: BillingPageBodyProps) {
  if (isLoading || error) {
    return (
      <FactorySettingsCard>
        <p className={error ? "text-sm text-destructive" : "text-sm text-muted-foreground"}>
          {error ? getApiErrorMessage(error, "Unable to load billing.") : "Loading billing..."}
        </p>
      </FactorySettingsCard>
    );
  }

  const showInvoices = billingEnabled && canManageBilling && hasBillingCustomer;

  return (
    <>
      {newBilling ? (
        <BillingPaymentSettings
          organizationId={organizationId}
          canManageBilling={canManageBilling}
          subscriptionPending={subscriptionPending}
          onCancelSubscription={onCancelSubscription}
          onKeepSubscription={onKeepSubscription}
        />
      ) : (
        <BillingPlansSection
          showUpgrade
          planName={planName}
          planPriceCents={planPriceCents}
          planPerSeat={planPerSeat}
          canManageBilling={canManageBilling}
          creditPurchaseAllowed={creditPurchaseAllowed}
          planSource={planSource}
          cancelAtPeriodEnd={cancelAtPeriodEnd}
          currentPeriodEnd={currentPeriodEnd}
          pending={businessCheckoutPending || subscriptionPending}
          onSubscribe={onSubscribe}
          onCancel={onCancelSubscription}
          onKeep={onKeepSubscription}
        />
      )}
      {seatsCard}
      <HostedCreditRemainingCard
        newBilling={newBilling}
        organizationId={organizationId}
        billingContactMessage={billingContactMessage}
        billingEnabled={billingEnabled}
        canManageBilling={canManageBilling}
        checkoutPending={checkoutPending}
        creditPurchaseAllowed={creditPurchaseAllowed}
        creditRefreshStatus={creditRefreshStatus}
        currentPeriodEnd={currentPeriodEnd}
        hasBillingCustomer={hasBillingCustomer}
        includedRemaining={includedRemaining}
        packs={packs}
        plan={plan}
        planName={planName}
        planCategory={planCategory}
        purchased={purchased}
        purchasedRemaining={purchasedRemaining}
        remaining={remaining}
        trialEndsAt={trialEndsAt}
        welcomeCreditExpiresAt={welcomeCreditExpiresAt}
        welcomeRemaining={welcomeRemaining}
        adminRemaining={adminRemaining}
        grants={grants}
        onAddCredit={onAddCredit}
      />
      <BillingHistory
        newBilling={newBilling}
        showInvoices={showInvoices}
        invoices={invoices}
        grants={grants}
        portalPending={portalPending}
        onManageInvoices={onManageInvoices}
      />
    </>
  );
}
