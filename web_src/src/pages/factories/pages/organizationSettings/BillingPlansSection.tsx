import { useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SUPERPLANE_PRICING_URL } from "@/lib/pricing";

import {
  BILLING_BUSINESS_DESCRIPTION,
  BILLING_BUSINESS_NAME,
  BILLING_BUSINESS_PRICE,
  BILLING_BUSINESS_PRICE_PERIOD,
  BILLING_CANCEL_LABEL,
  BILLING_KEEP_LABEL,
  BILLING_UPGRADE_LABEL,
  billingBusinessPlanAction,
  billingCanCancelBusiness,
  billingCanKeepBusiness,
  billingSubscriptionEndsCopy,
} from "../../lib/billingPlans";
import { FactorySettingsCard } from "../settings/FactorySettingsCard";
import { BillingCancelBusinessDialog } from "./BillingCancelBusinessDialog";

export function BillingPlansSection({
  canManageBilling,
  creditPurchaseAllowed,
  planSource,
  cancelAtPeriodEnd,
  currentPeriodEnd,
  pending,
  onSubscribe,
  onCancel,
  onKeep,
}: {
  canManageBilling: boolean;
  creditPurchaseAllowed: boolean;
  planSource?: string;
  cancelAtPeriodEnd?: boolean;
  currentPeriodEnd?: string;
  pending: boolean;
  onSubscribe: () => void | Promise<void>;
  onCancel?: () => void | Promise<void>;
  onKeep?: () => void | Promise<void>;
}) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const businessAction = billingBusinessPlanAction({
    canManageBilling,
    creditPurchaseAllowed,
    cancelAtPeriodEnd,
  });
  const showCurrentBadge = businessAction === "current" || businessAction === "ending";
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
  const endsCopy = businessAction === "ending" ? billingSubscriptionEndsCopy(currentPeriodEnd) : null;

  return (
    <FactorySettingsCard title="Plans" data-testid="billing-plans">
      <div data-testid="billing-plan-business">
        <div className="flex flex-wrap items-center gap-2">
          <h3 className="text-[13px] font-medium tracking-[-0.01em]">{BILLING_BUSINESS_NAME}</h3>
          {showCurrentBadge ? (
            <Badge variant="outline" className="text-muted-foreground" data-testid="billing-current-plan">
              Current plan
            </Badge>
          ) : null}
        </div>
        <p className="mt-2 flex flex-wrap items-baseline gap-x-1.5">
          <span className="text-xl font-semibold tracking-[-0.02em]">{BILLING_BUSINESS_PRICE}</span>
          <span className="text-xs text-muted-foreground">{BILLING_BUSINESS_PRICE_PERIOD}</span>
        </p>
        <p className="mt-1 text-sm text-muted-foreground">{BILLING_BUSINESS_DESCRIPTION}</p>
        {endsCopy ? (
          <p className="mt-2 text-sm text-muted-foreground" data-testid="billing-subscription-ends">
            {endsCopy}
          </p>
        ) : null}
        {businessAction === "subscribe" ? (
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
              onClick={() => setConfirmOpen(true)}
            >
              {BILLING_CANCEL_LABEL}
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
              {pending ? "Keeping Business..." : BILLING_KEEP_LABEL}
            </Button>
          </div>
        ) : null}
      </div>
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
        open={confirmOpen}
        pending={pending}
        currentPeriodEnd={currentPeriodEnd}
        onOpenChange={setConfirmOpen}
        onConfirm={onCancel ?? (async () => undefined)}
      />
    </FactorySettingsCard>
  );
}
