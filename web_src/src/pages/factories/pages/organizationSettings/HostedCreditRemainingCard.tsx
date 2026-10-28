import type {
  OrganizationsCreateHostedCreditCheckoutBody,
  OrganizationsHostedCreditProduct,
  OrganizationsOrganizationCreditGrant,
} from "@/api-client";
import { hostedCreditRefreshMessage, type HostedCreditRefreshStatus } from "@/lib/hostedCredit";
import { cn } from "@/lib/utils";
import { FactorySettingsCard } from "../settings/FactorySettingsCard";
import { HostedCreditTopUpBanner } from "./HostedCreditTopUpBanner";
import { billingPeriodEndLabel } from "../../lib/billingPlans";
import {
  adminCreditGrantCents,
  billingCreditBucketsView,
  billingCreditRemainingShares,
  type BillingCreditBarShare,
  type BillingCreditBucketKey,
  type BillingCreditBucketView,
} from "../../lib/billingCreditBuckets";
import { formatUsdCents } from "../../lib/workOrderUsage";
const HOSTED_CREDIT_EXPLANATION =
  "Hosted credit pays SuperPlane-hosted machines and managed models for this organization.";
const HOSTED_CREDIT_REMAINING_CAPTION = "Remaining hosted credit";

type HostedCreditRemainingCardProps = {
  organizationId: string;
  billingContactMessage?: string;
  canManageBilling: boolean;
  checkoutPending: boolean;
  creditPurchaseAllowed: boolean;
  creditRefreshStatus: HostedCreditRefreshStatus;
  currentPeriodEnd?: string;
  includedRemaining: number;
  packs: OrganizationsHostedCreditProduct[];
  plan?: string;
  planName?: string;
  planCategory?: string;
  purchased: number;
  purchasedRemaining: number;
  remaining: number;
  trialEndsAt?: string;
  welcomeCreditExpiresAt?: string;
  welcomeRemaining: number;
  adminRemaining: number;
  grants: OrganizationsOrganizationCreditGrant[];
  onAddCredit: (input: OrganizationsCreateHostedCreditCheckoutBody) => Promise<void>;
};

export function HostedCreditRemainingCard({
  organizationId,
  billingContactMessage,
  canManageBilling,
  checkoutPending,
  creditPurchaseAllowed,
  creditRefreshStatus,
  currentPeriodEnd,
  includedRemaining,
  packs,
  plan,
  planName,
  planCategory,
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
  const buckets = billingCreditBucketsView({
    welcomeRemainingCents: welcomeRemaining,
    includedRemainingCents: includedRemaining,
    purchasedRemainingCents: purchasedRemaining,
    purchasedCents: purchased,
    adminRemainingCents: adminRemaining,
    adminGrantCents: adminCreditGrantCents(grants),
    plan,
    planName,
    planCategory,
    trialEndsAt,
    welcomeCreditExpiresAt,
    currentPeriodEnd,
  });
  const description = remaining <= 0 ? "Your hosted credit balance is empty." : null;
  const creditRefreshMessage = hostedCreditRefreshMessage(creditRefreshStatus);
  const showBuyMore = canManageBilling && creditPurchaseAllowed;
  const visibleBuckets = buckets.filter((bucket) => bucket.remainingCents > 0);
  const remainingShares = billingCreditRemainingShares(visibleBuckets);

  return (
    <FactorySettingsCard
      title="Hosted credit"
      data-testid="billing-credit-balance"
      className="p-5 sm:p-6"
      action={
        showBuyMore && (
          <HostedCreditTopUpBanner
            organizationId={organizationId}
            checkoutPending={checkoutPending}
            packs={packs}
            onAddCredit={onAddCredit}
          />
        )
      }
    >
      <p className="text-[12px] text-muted-foreground">{HOSTED_CREDIT_EXPLANATION}</p>
      <div className="mt-4">
        <div className="flex flex-wrap items-center gap-2">
          <p
            className="text-3xl font-semibold tracking-[-0.02em] tabular-nums"
            data-testid="billing-credit-remaining-total"
          >
            {formatUsdCents(remaining)}
          </p>
        </div>
        <p className="mt-0.5 text-xs text-muted-foreground">{HOSTED_CREDIT_REMAINING_CAPTION}</p>
        <RemainingSharesBar shares={remainingShares} />
        <p className="mt-2 text-xs text-muted-foreground" data-testid="billing-credit-spend-order">
          {visibleBuckets.length > 1 ? "Credits are used in the order shown below." : ""}
        </p>
      </div>
      <CreditBalances
        visibleBuckets={visibleBuckets}
        expiresAt={trialEndsAt ?? welcomeCreditExpiresAt}
        currentPeriodEnd={currentPeriodEnd}
      />
      <CreditBalanceFeedback
        creditRefreshMessage={creditRefreshMessage}
        creditRefreshStatus={creditRefreshStatus}
        description={description}
        remaining={remaining}
        canManageBilling={canManageBilling}
        billingContactMessage={billingContactMessage}
      />
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

function creditRefreshClassName(status: HostedCreditRefreshStatus) {
  if (status === "added") {
    return "text-emerald-700 dark:text-emerald-400";
  }
  return "text-muted-foreground";
}

function BillingCreditBalanceRow({
  bucket,
  expiresAt,
  currentPeriodEnd,
}: {
  bucket: BillingCreditBucketView;
  expiresAt?: string;
  currentPeriodEnd?: string;
}) {
  const expiry = billingPeriodEndLabel(expiresAt);
  const renewal = billingPeriodEndLabel(currentPeriodEnd);
  const detail =
    bucket.key === "trial" && expiry
      ? `Expires on ${expiry}`
      : bucket.key === "included" && renewal
        ? `Resets on ${renewal}`
        : null;
  return (
    <li className="flex items-start justify-between gap-3 py-3 last:pb-0" data-testid={`billing-credit-${bucket.key}`}>
      <div>
        <div className="flex items-center gap-2">
          <span className={cn("size-1.5 rounded-full", bucketAccentClassName(bucket.key))} aria-hidden />
          <h3 className="text-sm">{bucket.key === "trial" ? "Signup credit" : bucket.heading}</h3>
        </div>
        {detail && <p className="mt-1 pl-3.5 text-xs text-muted-foreground">{detail}</p>}
      </div>
      <p className="text-sm font-medium tabular-nums" data-testid={`billing-credit-${bucket.key}-remaining`}>
        {formatUsdCents(bucket.remainingCents)}
      </p>
    </li>
  );
}

function CreditBalanceFeedback({
  creditRefreshMessage,
  creditRefreshStatus,
  description,
  remaining,
  canManageBilling,
  billingContactMessage,
}: Pick<
  HostedCreditRemainingCardProps,
  "creditRefreshStatus" | "remaining" | "canManageBilling" | "billingContactMessage"
> & {
  creditRefreshMessage: string | null;
  description: string | null;
}) {
  return (
    <>
      {creditRefreshMessage ? (
        <p className={`mt-3 text-sm ${creditRefreshClassName(creditRefreshStatus)}`}>{creditRefreshMessage}</p>
      ) : null}
      {description ? (
        <p
          className={cn(
            "mt-3 text-sm",
            remaining <= 0 ? "text-amber-700 dark:text-amber-400" : "text-muted-foreground",
          )}
        >
          {description}
        </p>
      ) : null}
      {!canManageBilling && billingContactMessage ? (
        <p className="mt-3 text-sm text-muted-foreground">{billingContactMessage}</p>
      ) : null}
    </>
  );
}

function CreditBalances({
  visibleBuckets,
  expiresAt,
  currentPeriodEnd,
}: {
  visibleBuckets: BillingCreditBucketView[];
  expiresAt?: string;
  currentPeriodEnd?: string;
}) {
  return (
    <ul className="mt-4 divide-y divide-border">
      {visibleBuckets.map((bucket) => (
        <BillingCreditBalanceRow
          key={bucket.key}
          bucket={bucket}
          expiresAt={expiresAt}
          currentPeriodEnd={currentPeriodEnd}
        />
      ))}
    </ul>
  );
}
