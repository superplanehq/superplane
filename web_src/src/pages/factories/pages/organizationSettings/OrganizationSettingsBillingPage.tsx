import { ChevronRight } from "lucide-react";
import { useState, type ReactNode } from "react";
import { useParams } from "react-router";

import type {
  OrganizationsHostedCreditInvoice,
  OrganizationsHostedCreditProduct,
  OrganizationsOrganizationCreditGrant,
} from "@/api-client";
import { Badge } from "@/components/ui/badge";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { hostedCreditRefreshMessage, type HostedCreditRefreshStatus } from "@/lib/hostedCredit";
import { cn } from "@/lib/utils";

import {
  adminCreditGrantCents,
  billingCreditBucketsView,
  billingCreditRemainingShares,
  billingSpendOrderCopy,
  type BillingCreditBarShare,
  type BillingCreditBucketKey,
  type BillingCreditBucketView,
} from "../../lib/billingCreditBuckets";
import { hostedCreditBillingBalanceCopy } from "../../lib/hostedCreditEmpty";
import { formatUsdCents, parseWorkOrderMetric } from "../../lib/workOrderUsage";
import { FactorySettingsCard, FactorySettingsPageFrame } from "../settings/FactorySettingsCard";
import { BillingCreditHistoryCard } from "./BillingCreditHistoryCard";
import { BillingInvoicesCard } from "./BillingInvoicesCard";
import { BillingPlansSection } from "./BillingPlansSection";
import { useOrganizationBillingPageModel } from "./useOrganizationBillingPageModel";

const BUY_MORE_PACK_CENTS = [5_000, 10_000, 50_000] as const;
const HOSTED_CREDIT_EXPLANATION =
  "Hosted credit pays SuperPlane-hosted machines and managed models for this organization.";
const HOSTED_CREDIT_REMAINING_CAPTION = "Remaining hosted credit";

export function OrganizationSettingsBillingPage() {
  const { organizationId = "" } = useParams<{ organizationId: string }>();
  const model = useOrganizationBillingPageModel(organizationId);
  const packs = model.billing.products;

  usePageTitle(["Billing", model.organizationName]);
  useReportPageReady(!model.isLoading, { failed: Boolean(model.error) });

  return (
    <FactorySettingsPageFrame title="Billing">
      <BillingPageBody
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
        onManageInvoices={model.billing.openInvoices}
      />
    </FactorySettingsPageFrame>
  );
}

function BillingPageBody({
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
  onManageInvoices,
}: {
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
  onAddCredit: (productId: string) => void | Promise<void>;
  onSubscribe: () => void | Promise<void>;
  onManageInvoices: () => void | Promise<void>;
}) {
  if (isLoading) {
    return (
      <FactorySettingsCard>
        <p className="text-sm text-muted-foreground">Loading billing...</p>
      </FactorySettingsCard>
    );
  }

  if (error) {
    return (
      <FactorySettingsCard>
        <p className="text-sm text-destructive">{getApiErrorMessage(error, "Unable to load billing.")}</p>
      </FactorySettingsCard>
    );
  }

  const showInvoices = billingEnabled && canManageBilling && hasBillingCustomer;

  return (
    <>
      <BillingPlansSection
        canManageBilling={canManageBilling}
        creditPurchaseAllowed={creditPurchaseAllowed}
        pending={businessCheckoutPending}
        onSubscribe={onSubscribe}
      />
      <HostedCreditRemainingCard
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
      {showInvoices ? (
        <BillingInvoicesCard invoices={invoices} portalPending={portalPending} onManageInvoices={onManageInvoices} />
      ) : null}
      <BillingCreditHistoryCard grants={grants} />
    </>
  );
}

type HostedCreditRemainingCardProps = {
  billingContactMessage?: string;
  billingEnabled: boolean;
  canManageBilling: boolean;
  checkoutPending: boolean;
  creditPurchaseAllowed: boolean;
  creditRefreshStatus: HostedCreditRefreshStatus;
  currentPeriodEnd?: string;
  hasBillingCustomer: boolean;
  includedRemaining: number;
  packs: OrganizationsHostedCreditProduct[];
  plan?: string;
  purchased: number;
  purchasedRemaining: number;
  remaining: number;
  trialEndsAt?: string;
  welcomeCreditExpiresAt?: string;
  welcomeRemaining: number;
  adminRemaining: number;
  grants: OrganizationsOrganizationCreditGrant[];
  onAddCredit: (productId: string) => void | Promise<void>;
};

function HostedCreditRemainingCard({
  billingContactMessage,
  billingEnabled,
  canManageBilling,
  checkoutPending,
  creditPurchaseAllowed,
  creditRefreshStatus,
  currentPeriodEnd,
  hasBillingCustomer,
  includedRemaining,
  packs,
  plan,
  purchased,
  purchasedRemaining,
  remaining,
  trialEndsAt,
  welcomeCreditExpiresAt,
  welcomeRemaining,
  adminRemaining,
  grants,
  onAddCredit,
}: HostedCreditRemainingCardProps) {
  const copy = hostedCreditBillingBalanceCopy({
    remainingCents: remaining,
    purchasedCents: purchased,
    hasBillingCustomer,
    billingEnabled,
    welcomeCreditExpiresAt,
    plan,
    trialEndsAt,
    creditPurchaseAllowed,
  });
  const buckets = billingCreditBucketsView({
    welcomeRemainingCents: welcomeRemaining,
    includedRemainingCents: includedRemaining,
    purchasedRemainingCents: purchasedRemaining,
    purchasedCents: purchased,
    adminRemainingCents: adminRemaining,
    adminGrantCents: adminCreditGrantCents(grants),
    plan,
    trialEndsAt,
    welcomeCreditExpiresAt,
    currentPeriodEnd,
  });
  const creditRefreshMessage = hostedCreditRefreshMessage(creditRefreshStatus);
  const showBuyMore = canManageBilling && creditPurchaseAllowed;
  const remainingShares = billingCreditRemainingShares(buckets);
  const spendOrderCopy = billingSpendOrderCopy(buckets.some((bucket) => bucket.key === "grant"));

  return (
    <FactorySettingsCard title="Hosted credit" data-testid="billing-credit-balance">
      <p className="text-[12px] text-muted-foreground">{HOSTED_CREDIT_EXPLANATION}</p>
      <div className="mt-4">
        <div className="flex flex-wrap items-center gap-2">
          <p
            className="text-xl font-semibold tracking-[-0.02em] tabular-nums"
            data-testid="billing-credit-remaining-total"
          >
            {formatUsdCents(remaining)}
          </p>
          {copy.badge === "Trial" ? (
            <Badge variant="outline" className="text-muted-foreground">
              {copy.badge}
            </Badge>
          ) : null}
        </div>
        <p className="mt-0.5 text-xs text-muted-foreground">{HOSTED_CREDIT_REMAINING_CAPTION}</p>
        <RemainingSharesBar shares={remainingShares} />
        <p className="mt-2 text-xs text-muted-foreground" data-testid="billing-credit-spend-order">
          {spendOrderCopy}
        </p>
      </div>
      <ul className="mt-4 divide-y divide-border">
        {buckets.map((bucket) => (
          <CreditBucketRow
            key={bucket.key}
            bucket={bucket}
            action={
              bucket.key === "topup" && showBuyMore ? (
                <HostedCreditTopUpBanner checkoutPending={checkoutPending} packs={packs} onAddCredit={onAddCredit} />
              ) : undefined
            }
          />
        ))}
      </ul>
      {creditRefreshMessage ? (
        <p className={`mt-3 text-sm ${creditRefreshClassName(creditRefreshStatus)}`}>{creditRefreshMessage}</p>
      ) : null}
      {copy.description ? (
        <p
          className={cn(
            "mt-3 text-sm",
            remaining <= 0 ? "text-amber-700 dark:text-amber-400" : "text-muted-foreground",
          )}
        >
          {copy.description}
        </p>
      ) : null}
      {!canManageBilling && billingContactMessage ? (
        <p className="mt-3 text-sm text-muted-foreground">{billingContactMessage}</p>
      ) : null}
    </FactorySettingsCard>
  );
}

function RemainingSharesBar({ shares }: { shares: BillingCreditBarShare[] }) {
  const hasRemaining = shares.some((share) => share.percent > 0);

  return (
    <div
      className="mt-3 flex h-2 overflow-hidden rounded-full bg-muted"
      data-testid="billing-credit-remaining-shares"
      role="img"
      aria-label="Remaining credit by spend order"
    >
      {hasRemaining
        ? shares
            .filter((share) => share.percent > 0)
            .map((share) => (
              <div
                key={share.key}
                className={cn("h-full", bucketAccentClassName(share.key))}
                style={{ width: `${share.percent}%` }}
              />
            ))
        : null}
    </div>
  );
}

function CreditBucketRow({ bucket, action }: { bucket: BillingCreditBucketView; action?: ReactNode }) {
  const remainingPercent = Math.max(0, 100 - bucket.usedPercent);

  return (
    <li className="py-3 first:pt-1 last:pb-0" data-testid={`billing-credit-${bucket.key}`}>
      <div className="flex items-baseline justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className={cn("size-1.5 shrink-0 rounded-full", bucketAccentClassName(bucket.key))} aria-hidden />
            <h3 className="text-[13px] font-medium tracking-[-0.01em]">{bucket.heading}</h3>
          </div>
          <p className="mt-0.5 pl-3.5 text-[11px] text-muted-foreground">{bucket.spendOrderLabel}</p>
        </div>
        <p
          className="shrink-0 text-[13px] font-medium tracking-[-0.01em] tabular-nums"
          data-testid={`billing-credit-${bucket.key}-remaining`}
        >
          {bucket.remainingLabel}
        </p>
      </div>
      <div
        className="mt-2 h-1.5 overflow-hidden rounded-full bg-muted"
        role="progressbar"
        aria-label={`${bucket.heading} remaining`}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={remainingPercent}
        aria-valuetext={bucket.remainingLabel}
      >
        <div
          className={cn("h-full rounded-full", bucketAccentClassName(bucket.key))}
          style={{ width: `${remainingPercent}%` }}
        />
      </div>
      {bucket.footer ? <p className="mt-1.5 text-xs text-muted-foreground">{bucket.footer}</p> : null}
      {action}
    </li>
  );
}

function bucketAccentClassName(key: BillingCreditBucketKey) {
  switch (key) {
    case "trial":
      return "bg-amber-500";
    case "included":
      return "bg-foreground";
    case "topup":
      return "bg-sky-500";
    case "grant":
      return "bg-emerald-500";
  }
}

function HostedCreditTopUpBanner({
  checkoutPending,
  packs,
  onAddCredit,
}: {
  checkoutPending: boolean;
  packs: OrganizationsHostedCreditProduct[];
  onAddCredit: (productId: string) => void | Promise<void>;
}) {
  const [open, setOpen] = useState(false);
  const hasPurchasablePack = BUY_MORE_PACK_CENTS.some((cents) => findPackForCents(packs, cents));
  const disabled = checkoutPending || !hasPurchasablePack;

  return (
    <div className="mt-3 inline-flex h-8 items-center gap-1.5 rounded-full bg-violet-100 py-1 pr-1.5 pl-2.5 text-[12px] hover:bg-violet-200/80 dark:bg-violet-950 dark:hover:bg-violet-900">
      <button
        type="button"
        disabled={disabled}
        aria-expanded={open}
        aria-label="Top up"
        data-testid="billing-top-up"
        onClick={() => setOpen((current) => !current)}
        className="inline-flex items-center gap-1 whitespace-nowrap font-medium text-violet-800 disabled:pointer-events-none disabled:opacity-50 dark:text-violet-200"
      >
        {checkoutPending ? "Opening checkout..." : "Top up"}
        <ChevronRight className={cn("size-3.5 transition-transform", open && "rotate-180")} aria-hidden />
      </button>
      {open ? (
        <div className="flex items-center gap-1" data-testid="billing-top-up-options">
          {BUY_MORE_PACK_CENTS.map((cents) => {
            const product = findPackForCents(packs, cents);
            const productId = product?.id ?? "";
            return (
              <button
                key={cents}
                type="button"
                disabled={!productId || checkoutPending}
                onClick={() => productId && void onAddCredit(productId)}
                className="inline-flex h-5 items-center rounded-full bg-violet-600 px-2.5 text-[11px] leading-none font-medium text-white hover:bg-violet-700 disabled:pointer-events-none disabled:opacity-40"
              >
                {formatUsdPackLabel(cents)}
              </button>
            );
          })}
          <button
            type="button"
            disabled
            title="Custom amounts are not available yet."
            className="inline-flex h-5 cursor-not-allowed items-center rounded-full bg-violet-600/40 px-2.5 text-[11px] leading-none font-medium text-white"
          >
            Custom
          </button>
        </div>
      ) : null}
    </div>
  );
}

function findPackForCents(packs: OrganizationsHostedCreditProduct[], cents: number) {
  return packs.find((product) => Boolean(product.id) && parseWorkOrderMetric(product.amountCents) === cents);
}

function formatUsdPackLabel(cents: number): string {
  const dollars = cents / 100;
  if (Number.isInteger(dollars)) {
    return `$${dollars}`;
  }
  return formatUsdCents(cents);
}

function creditRefreshClassName(status: HostedCreditRefreshStatus) {
  if (status === "added") {
    return "text-emerald-700 dark:text-emerald-400";
  }
  return "text-muted-foreground";
}
