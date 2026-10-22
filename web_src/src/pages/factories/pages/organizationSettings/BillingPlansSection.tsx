import { useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SUPERPLANE_PRICING_URL } from "@/lib/pricing";

import {
  BILLING_BUSINESS_DESCRIPTION,
  BILLING_BUSINESS_NAME,
  BILLING_BUSINESS_PRICE,
  BILLING_BUSINESS_PRICE_PERIOD,
  BILLING_UPGRADE_LABEL,
  billingBusinessPlanAction,
  billingCanCancelBusiness,
  billingCanKeepBusiness,
  billingSubscriptionEndsCopy,
} from "../../lib/billingPlans";
import { formatUsdCents, parseWorkOrderMetric } from "../../lib/workOrderUsage";
import { FactorySettingsCard } from "../settings/FactorySettingsCard";
import { BillingCancelBusinessDialog } from "./BillingCancelBusinessDialog";

type BillingPlansSectionProps = {
  showUpgrade?: boolean;
  planName?: string;
  planPriceCents?: string;
  planPerSeat?: boolean;
  canManageBilling: boolean;
  creditPurchaseAllowed: boolean;
  planSource?: string;
  cancelAtPeriodEnd?: boolean;
  currentPeriodEnd?: string;
  pending: boolean;
  onSubscribe: () => void | Promise<void>;
  onCancel?: () => void | Promise<void>;
  onKeep?: () => void | Promise<void>;
};

export function BillingPlansSection({
  showUpgrade = true,
  planName,
  planPriceCents,
  planPerSeat,
  canManageBilling,
  creditPurchaseAllowed,
  planSource,
  cancelAtPeriodEnd,
  currentPeriodEnd,
  pending,
  onSubscribe,
  onCancel,
  onKeep,
}: BillingPlansSectionProps) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const businessAction = billingBusinessPlanAction({
    canManageBilling,
    creditPurchaseAllowed,
    cancelAtPeriodEnd,
  });
  const showCurrentBadge = !showUpgrade || businessAction === "current" || businessAction === "ending";
  const displayName = showCurrentBadge ? planName || BILLING_BUSINESS_NAME : BILLING_BUSINESS_NAME;
  const endsCopy = businessAction === "ending" ? billingSubscriptionEndsCopy(currentPeriodEnd, displayName) : null;

  return (
    <FactorySettingsCard title="Plans" data-testid="billing-plans">
      <div data-testid="billing-plan-business">
        <BillingPlanDetails
          displayName={displayName}
          planName={planName}
          planPriceCents={planPriceCents}
          planPerSeat={planPerSeat}
          showCurrentBadge={showCurrentBadge}
        />
        {endsCopy ? (
          <p className="mt-2 text-sm text-muted-foreground" data-testid="billing-subscription-ends">
            {endsCopy}
          </p>
        ) : null}
        <BillingPlanActions
          showUpgrade={showUpgrade}
          businessAction={businessAction}
          canManageBilling={canManageBilling}
          creditPurchaseAllowed={creditPurchaseAllowed}
          planSource={planSource}
          cancelAtPeriodEnd={cancelAtPeriodEnd}
          pending={pending}
          displayName={displayName}
          onSubscribe={onSubscribe}
          onCancel={onCancel}
          onKeep={onKeep}
          onRequestCancel={() => setConfirmOpen(true)}
        />
      </div>
      {!showUpgrade ? (
        <p className="mt-3 text-sm text-muted-foreground">
          Select an available plan in Payment method. Contact an installation admin if the plan you need is unavailable.
        </p>
      ) : null}
      <p className="mt-3 text-xs text-muted-foreground">
        For air-gapped installation, audit logs, or invoice billing,{" "}
        <a
          href={SUPERPLANE_PRICING_URL}
          target="_blank"
          rel="noreferrer"
          className="underline underline-offset-2"
          data-testid="billing-talk-to-us"
        >
          Talk to us
        </a>
      </p>
      <BillingCancelBusinessDialog
        planName={displayName}
        open={confirmOpen}
        pending={pending}
        currentPeriodEnd={currentPeriodEnd}
        onOpenChange={setConfirmOpen}
        onConfirm={onCancel ?? (async () => undefined)}
      />
    </FactorySettingsCard>
  );
}

function BillingPlanActions({
  showUpgrade,
  businessAction,
  canManageBilling,
  creditPurchaseAllowed,
  planSource,
  cancelAtPeriodEnd,
  pending,
  displayName,
  onSubscribe,
  onCancel,
  onKeep,
  onRequestCancel,
}: BillingPlansSectionProps & {
  businessAction: ReturnType<typeof billingBusinessPlanAction>;
  displayName: string;
  onRequestCancel: () => void;
}) {
  const canCancel = billingCanCancelBusiness({
    canManageBilling,
    creditPurchaseAllowed,
    planSource,
    cancelAtPeriodEnd,
  });
  const canKeep = billingCanKeepBusiness({
    canManageBilling,
    creditPurchaseAllowed,
    planSource,
    cancelAtPeriodEnd,
  });
  return (
    <>
      {showUpgrade && businessAction === "subscribe" ? (
        <div className="mt-4">
          <Button type="button" disabled={pending} data-testid="billing-subscribe" onClick={() => void onSubscribe()}>
            {pending ? "Opening checkout..." : BILLING_UPGRADE_LABEL}
          </Button>
        </div>
      ) : null}
      {canCancel && onCancel ? (
        <div className="mt-4">
          <Button
            type="button"
            variant="outline"
            disabled={pending}
            data-testid="billing-cancel-subscription"
            onClick={onRequestCancel}
          >
            {`Cancel ${displayName}`}
          </Button>
        </div>
      ) : null}
      {canKeep && onKeep ? (
        <div className="mt-4">
          <Button
            type="button"
            disabled={pending}
            data-testid="billing-keep-subscription"
            onClick={() => void onKeep()}
          >
            {pending ? `Keeping ${displayName}...` : `Keep ${displayName}`}
          </Button>
        </div>
      ) : null}
    </>
  );
}

function BillingPlanDetails({
  displayName,
  planName,
  planPriceCents,
  planPerSeat,
  showCurrentBadge,
}: {
  displayName: string;
  planName?: string;
  planPriceCents?: string;
  planPerSeat?: boolean;
  showCurrentBadge: boolean;
}) {
  const displayPrice =
    showCurrentBadge && planPriceCents !== undefined
      ? formatUsdCents(parseWorkOrderMetric(planPriceCents))
      : BILLING_BUSINESS_PRICE;

  return (
    <>
      {!showCurrentBadge && planName ? (
        <div className="mb-4 flex flex-wrap items-center gap-2">
          <h3 className="text-[13px] font-medium tracking-[-0.01em]">{planName}</h3>
          <Badge variant="outline" className="text-muted-foreground" data-testid="billing-current-plan">
            Current plan
          </Badge>
        </div>
      ) : null}
      <div className="flex flex-wrap items-center gap-2">
        <h3 className="text-[13px] font-medium tracking-[-0.01em]">{displayName}</h3>
        {showCurrentBadge ? (
          <Badge variant="outline" className="text-muted-foreground" data-testid="billing-current-plan">
            Current plan
          </Badge>
        ) : null}
      </div>
      <p className="mt-2 flex flex-wrap items-baseline gap-x-1.5">
        <span className="text-xl font-semibold tracking-[-0.02em]">{displayPrice}</span>
        <span className="text-xs text-muted-foreground">
          {showCurrentBadge && planPerSeat ? "per seat per month" : BILLING_BUSINESS_PRICE_PERIOD}
        </span>
      </p>
      {displayName === BILLING_BUSINESS_NAME ? (
        <p className="mt-1 text-sm text-muted-foreground">{BILLING_BUSINESS_DESCRIPTION}</p>
      ) : null}
    </>
  );
}
