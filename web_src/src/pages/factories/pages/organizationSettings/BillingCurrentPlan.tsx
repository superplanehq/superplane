import { CreditCard, ExternalLink } from "lucide-react";
import { useState, type Ref } from "react";
import type { OrganizationsDescribeOrganizationBillingResponse } from "@/api-client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  billingPlanParallelism,
  billingCanCancelBusiness,
  billingCanKeepBusiness,
  billingIsPastDue,
  billingPaymentBlocked,
  billingPeriodEndLabel,
  billingSubscriptionEndsCopy,
} from "../../lib/billingPlans";
import { formatUsdCents } from "../../lib/workOrderUsage";
import { FactorySettingsCard } from "../settings/FactorySettingsCard";
import { BillingCancelBusinessDialog } from "./BillingCancelBusinessDialog";

type BillingCurrentPlanProps = {
  changePlanButtonRef: Ref<HTMLButtonElement>;
  billing: OrganizationsDescribeOrganizationBillingResponse;
  canManageBilling: boolean;
  busy: boolean;
  onChangePlan: () => void;
  onManagePayment: () => void;
  onCancel?: () => void | Promise<void>;
  onKeep?: () => void | Promise<void>;
};

export function BillingCurrentPlan(props: BillingCurrentPlanProps) {
  const { billing, canManageBilling, busy, onChangePlan } = props;
  const current = billing.availablePlans?.find((plan) => plan.key === billing.catalogKey);
  const name = billing.planName || current?.name || "No plan selected";
  const price = billing.planPriceCents ?? current?.priceCents;
  const perSeat = billing.planPerSeat ?? current?.perSeat;
  const canChange =
    canManageBilling &&
    !billingPaymentBlocked(billing.polarSubscriptionStatus) &&
    Boolean(billing.availablePlans?.length);
  return (
    <FactorySettingsCard
      title="Your plan"
      data-testid="billing-plans"
      className="overflow-hidden p-5 sm:p-6"
      action={
        canChange && (
          <Button ref={props.changePlanButtonRef} size="sm" variant="outline" disabled={busy} onClick={onChangePlan}>
            Change plan
          </Button>
        )
      }
    >
      <div className="flex flex-wrap items-start justify-between gap-4 pb-5">
        <div>
          <div className="flex flex-wrap items-center gap-2.5">
            <h3 className="text-xl font-semibold tracking-tight">{name}</h3>
            <Badge variant="outline" className="text-muted-foreground" data-testid="billing-current-plan">
              Current plan
            </Badge>
          </div>
          {current && (
            <p className="mt-2 text-sm text-muted-foreground">{billingPlanParallelism(current.parallelTaskLimit)}</p>
          )}
        </div>
        <PlanPrice price={price} perSeat={perSeat} />
      </div>
      <SubscriptionStatus
        billing={billing}
        name={name}
        canManageBilling={canManageBilling}
        busy={busy}
        onCancel={props.onCancel}
        onKeep={props.onKeep}
      />
      <PaymentMethodRow
        billing={billing}
        canManageBilling={canManageBilling}
        busy={busy}
        onManagePayment={props.onManagePayment}
      />
    </FactorySettingsCard>
  );
}

function SubscriptionStatus({
  billing,
  name,
  canManageBilling,
  busy,
  onCancel,
  onKeep,
}: Pick<BillingCurrentPlanProps, "billing" | "canManageBilling" | "busy" | "onCancel" | "onKeep"> & { name: string }) {
  const [cancelOpen, setCancelOpen] = useState(false);
  const subscription = { ...billing, canManageBilling, creditPurchaseAllowed: Boolean(billing.creditPurchaseAllowed) };
  const canKeep = billingCanKeepBusiness(subscription) && onKeep;
  const canCancel = billingCanCancelBusiness(subscription) && onCancel;
  return (
    <>
      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-border py-3 text-xs text-muted-foreground">
        <p data-testid={billing.cancelAtPeriodEnd ? "billing-subscription-ends" : undefined}>
          {subscriptionStatusCopy(billing, name)}
        </p>
        {canKeep && (
          <Button size="sm" variant="outline" disabled={busy} onClick={() => void onKeep()}>{`Keep ${name}`}</Button>
        )}
        {canCancel && (
          <Button
            size="sm"
            variant="ghost"
            className="h-auto p-0 text-xs text-muted-foreground"
            disabled={busy}
            onClick={() => setCancelOpen(true)}
          >{`Cancel ${name}`}</Button>
        )}
      </div>
      {onCancel && (
        <BillingCancelBusinessDialog
          planName={name}
          open={cancelOpen}
          pending={busy}
          currentPeriodEnd={billing.currentPeriodEnd}
          onOpenChange={setCancelOpen}
          onConfirm={onCancel}
        />
      )}
    </>
  );
}

function subscriptionStatusCopy(billing: OrganizationsDescribeOrganizationBillingResponse, name: string) {
  if (billing.cancelAtPeriodEnd)
    return (
      billingSubscriptionEndsCopy(billing.currentPeriodEnd, name) ||
      "Your subscription ends after the current billing period."
    );
  if (billingPaymentBlocked(billing.polarSubscriptionStatus)) return "Update your payment to restore access.";
  const periodEnd = billingPeriodEndLabel(billing.currentPeriodEnd);
  if (billing.planSource === "polar" && periodEnd) return `Renews on ${periodEnd}`;
  if (Number(billing.planPriceCents) === 0) return "No subscription fee. Hosted usage is billed separately.";
  return "Hosted usage is billed separately.";
}

function PaymentMethodRow({
  billing,
  canManageBilling,
  busy,
  onManagePayment,
}: Pick<BillingCurrentPlanProps, "billing" | "canManageBilling" | "busy" | "onManagePayment">) {
  const hasPortal =
    billing.hasPaymentMethod ||
    billing.planSource === "polar" ||
    billingPaymentBlocked(billing.polarSubscriptionStatus);
  return (
    <div className="-mx-5 -mb-5 flex flex-wrap items-center justify-between gap-3 border-t border-border bg-muted/20 px-5 py-4 sm:-mx-6 sm:-mb-6 sm:px-6">
      <div className="flex items-center gap-3">
        <span className="flex size-9 items-center justify-center rounded-md border border-border bg-background">
          <CreditCard className="size-4 text-muted-foreground" aria-hidden />
        </span>
        <div>
          <h3 className="text-sm font-medium">Payment method</h3>
          <p className="mt-0.5 text-xs text-muted-foreground">
            {billing.hasPaymentMethod ? "Card saved · Managed securely by Polar" : "No card on file"}
          </p>
        </div>
      </div>
      {canManageBilling && !billingIsPastDue(billing.polarSubscriptionStatus) && (
        <Button size="sm" variant="ghost" disabled={busy} onClick={onManagePayment}>
          {hasPortal ? "Manage payment method" : "Add payment method"}
          {hasPortal && <ExternalLink className="size-3.5" aria-hidden />}
        </Button>
      )}
      {billing.requirePaymentMethod && !billing.hasPaymentMethod && (
        <p className="w-full text-xs text-amber-700 dark:text-amber-400">Add a card to start hosted tasks.</p>
      )}
      {!canManageBilling && (
        <p className="w-full text-xs text-muted-foreground">Ask an organization admin to manage billing.</p>
      )}
    </div>
  );
}

function PlanPrice({ price, perSeat }: { price?: string; perSeat?: boolean }) {
  if (price == null) return null;
  return (
    <div className="sm:text-right">
      <p className="text-2xl font-semibold tracking-tight tabular-nums">{formatUsdCents(Number(price))}</p>
      <p className="mt-1 text-xs text-muted-foreground">{perSeat ? "per seat per month" : "per month"}</p>
    </div>
  );
}
