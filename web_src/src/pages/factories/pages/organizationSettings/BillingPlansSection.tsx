import { useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SUPERPLANE_PRICING_URL } from "@/lib/pricing";

import {
  BILLING_BUSINESS_DESCRIPTION,
  BILLING_BUSINESS_NAME,
  BILLING_BUSINESS_PRICE,
  BILLING_BUSINESS_PRICE_PERIOD,
  billingBusinessPlanAction,
  billingCanCancelBusiness,
  billingCanKeepBusiness,
  billingSubscriptionEndsCopy,
} from "../../lib/billingPlans";
import { formatUsdCents, parseWorkOrderMetric } from "../../lib/workOrderUsage";
import { FactorySettingsCard } from "../settings/FactorySettingsCard";
import { BillingCancelBusinessDialog } from "./BillingCancelBusinessDialog";

type BillingPlansSectionProps = {
  planName?: string;
  planPriceCents?: string;
  planPerSeat?: boolean;
  canManageBilling: boolean;
  creditPurchaseAllowed: boolean;
  planSource?: string;
  cancelAtPeriodEnd?: boolean;
  currentPeriodEnd?: string;
  pending: boolean;
  onCancel?: () => void | Promise<void>;
  onKeep?: () => void | Promise<void>;
};

export function BillingPlansSection({
  planName,
  planPriceCents,
  planPerSeat,
  canManageBilling,
  creditPurchaseAllowed,
  planSource,
  cancelAtPeriodEnd,
  currentPeriodEnd,
  pending,
  onCancel,
  onKeep,
}: BillingPlansSectionProps) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const businessAction = billingBusinessPlanAction({
    canManageBilling,
    creditPurchaseAllowed,
    cancelAtPeriodEnd,
  });
  const displayName = planName || BILLING_BUSINESS_NAME;
  const endsCopy = businessAction === "ending" ? billingSubscriptionEndsCopy(currentPeriodEnd, displayName) : null;

  return (
    <FactorySettingsCard title="Plans" data-testid="billing-plans">
      <div data-testid="billing-plan-business">
        <BillingPlanDetails displayName={displayName} planPriceCents={planPriceCents} planPerSeat={planPerSeat} />
        {endsCopy ? (
          <p className="mt-2 text-sm text-muted-foreground" data-testid="billing-subscription-ends">
            {endsCopy}
          </p>
        ) : null}
        <BillingPlanActions
          canManageBilling={canManageBilling}
          creditPurchaseAllowed={creditPurchaseAllowed}
          planSource={planSource}
          cancelAtPeriodEnd={cancelAtPeriodEnd}
          pending={pending}
          displayName={displayName}
          onCancel={onCancel}
          onKeep={onKeep}
          onRequestCancel={() => setConfirmOpen(true)}
          planPerSeat={planPerSeat}
        />
      </div>
      <p className="mt-3 text-sm text-muted-foreground">
        Select an available plan in Payment method. Contact an installation admin if the plan you need is unavailable.
      </p>
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
  canManageBilling,
  creditPurchaseAllowed,
  planSource,
  cancelAtPeriodEnd,
  pending,
  displayName,
  planPerSeat,
  onCancel,
  onKeep,
  onRequestCancel,
}: BillingPlansSectionProps & {
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
        <div className="mt-4 space-y-2">
          {planPerSeat && <p className="text-sm text-muted-foreground">Resume the subscription before buying seats.</p>}
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
  planPriceCents,
  planPerSeat,
}: {
  displayName: string;
  planPriceCents?: string;
  planPerSeat?: boolean;
}) {
  const displayPrice =
    planPriceCents !== undefined ? formatUsdCents(parseWorkOrderMetric(planPriceCents)) : BILLING_BUSINESS_PRICE;

  return (
    <>
      <div className="flex flex-wrap items-center gap-2">
        <h3 className="text-[13px] font-medium tracking-[-0.01em]">{displayName}</h3>
        <Badge variant="outline" className="text-muted-foreground" data-testid="billing-current-plan">
          Current plan
        </Badge>
      </div>
      <p className="mt-2 flex flex-wrap items-baseline gap-x-1.5">
        <span className="text-xl font-semibold tracking-[-0.02em]">{displayPrice}</span>
        <span className="text-xs text-muted-foreground">
          {planPerSeat ? "per seat per month" : BILLING_BUSINESS_PRICE_PERIOD}
        </span>
      </p>
      {displayName === BILLING_BUSINESS_NAME ? (
        <p className="mt-1 text-sm text-muted-foreground">{BILLING_BUSINESS_DESCRIPTION}</p>
      ) : null}
    </>
  );
}
