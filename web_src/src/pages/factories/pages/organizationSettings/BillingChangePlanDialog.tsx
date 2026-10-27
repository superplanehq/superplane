import { ArrowLeft, Check, ExternalLink } from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";
import type { OrganizationsBillingPlanOption, OrganizationsDescribeOrganizationBillingResponse } from "@/api-client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { SUPERPLANE_PRICING_URL } from "@/lib/pricing";
import { cn } from "@/lib/utils";
import { formatUsdCents } from "../../lib/workOrderUsage";
import { billingPlanParallelism } from "../../lib/billingPlans";
import { BillingSeatCheckout } from "./BillingSeatCheckout";

export function BillingChangePlanDialog({
  open,
  onOpenChange,
  onCloseAutoFocus,
  billing,
  busy,
  subscriptionActive,
  onSelect,
  onSubscribeWithSeats,
  onManageSubscription,
  feedback,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCloseAutoFocus: () => void;
  billing: OrganizationsDescribeOrganizationBillingResponse;
  busy: boolean;
  subscriptionActive: boolean;
  onSelect: (plan: OrganizationsBillingPlanOption) => void;
  onSubscribeWithSeats: (planKey?: string, seats?: number) => void;
  onManageSubscription: () => void;
  feedback: ReactNode;
}) {
  const [seatPlanKey, setSeatPlanKey] = useState<string>();
  useEffect(() => {
    if (!open) setSeatPlanKey(undefined);
  }, [open]);
  const seatPlan = billing.availablePlans?.find((plan) => plan.key === seatPlanKey);
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (busy) return;
        onOpenChange(next);
        if (!next) setSeatPlanKey(undefined);
      }}
    >
      <DialogContent
        onCloseAutoFocus={(event) => {
          event.preventDefault();
          onCloseAutoFocus();
        }}
        size="large"
        className={cn("max-h-[90vh] w-[calc(100%-2rem)] overflow-y-auto", seatPlan ? "max-w-md" : "max-w-3xl")}
      >
        <DialogHeader>
          <DialogTitle>{seatPlan ? `Set up ${seatPlan.name}` : "Change plan"}</DialogTitle>
          <DialogDescription>
            {seatPlan
              ? "Choose seats for your organization. Review the final charge in checkout."
              : "Choose the capacity your organization needs. Hosted usage is billed separately."}
          </DialogDescription>
        </DialogHeader>
        {seatPlan ? (
          <>
            <Button
              variant="ghost"
              size="sm"
              className="w-fit px-0"
              disabled={busy}
              onClick={() => setSeatPlanKey(undefined)}
            >
              <ArrowLeft className="size-4" aria-hidden />
              Back to plans
            </Button>
            <p className="text-sm text-muted-foreground">
              {formatUsdCents(Number(seatPlan.priceCents ?? 0))} per seat per month
            </p>
            <BillingSeatCheckout
              plan={seatPlan}
              occupied={billing.occupiedSeats}
              pending={billing.pendingInvitationSeats}
              disabled={busy}
              onSubscribe={(seats) => onSubscribeWithSeats(seatPlan.key, seats)}
            />
          </>
        ) : (
          <>
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {(billing.availablePlans ?? []).map((plan) => (
                <PlanOption
                  key={plan.key}
                  plan={plan}
                  current={billing.catalogKey === plan.key}
                  needsCard={Boolean(
                    plan.requirePaymentMethod && !billing.hasPaymentMethod && !plan.subscriptionRequired,
                  )}
                  busy={busy}
                  subscriptionActive={subscriptionActive}
                  onSelect={() => (plan.perSeat ? setSeatPlanKey(plan.key) : onSelect(plan))}
                />
              ))}
            </div>
            {subscriptionActive && (
              <div className="flex flex-wrap items-center justify-between gap-3 rounded-md bg-muted/40 p-3">
                <p className="text-xs text-muted-foreground">Review plan changes and any charges in Polar.</p>
                <Button size="sm" disabled={busy} onClick={onManageSubscription}>
                  Manage subscription
                  <ExternalLink className="size-3.5" aria-hidden />
                </Button>
              </div>
            )}
            <p className="text-xs text-muted-foreground">
              Need audit logs, invoice billing, or a private installation?{" "}
              <a
                className="underline underline-offset-4"
                href={SUPERPLANE_PRICING_URL}
                target="_blank"
                rel="noreferrer"
              >
                Talk to us
              </a>
            </p>
          </>
        )}
        {feedback}
      </DialogContent>
    </Dialog>
  );
}

function PlanOption({
  plan,
  current,
  needsCard,
  busy,
  subscriptionActive,
  onSelect,
}: {
  plan: OrganizationsBillingPlanOption;
  current: boolean;
  needsCard: boolean;
  busy: boolean;
  subscriptionActive: boolean;
  onSelect: () => void;
}) {
  return (
    <div
      className={cn(
        "flex flex-col rounded-lg border p-4",
        current ? "border-foreground/30 bg-muted/30" : "border-border",
      )}
    >
      <div className="flex min-h-6 flex-wrap items-center justify-between gap-2">
        <h3 className="text-sm font-semibold">{plan.name}</h3>
        {current && (
          <Badge variant="outline" className="text-[10px]">
            Current
          </Badge>
        )}
      </div>
      <p className="mt-4 text-2xl font-semibold tracking-tight">{formatUsdCents(Number(plan.priceCents ?? 0))}</p>
      <p className="mt-1 text-xs text-muted-foreground">{plan.perSeat ? "per seat per month" : "per month"}</p>
      <ul className="my-5 flex-1 space-y-2 text-xs text-muted-foreground">
        <li className="flex items-start gap-2">
          <Check className="size-3.5 shrink-0" aria-hidden />
          {billingPlanParallelism(plan.parallelTaskLimit)}
        </li>
        {plan.seatMaximum != null && (
          <li className="flex items-start gap-2">
            <Check className="size-3.5 shrink-0" aria-hidden />
            Up to {plan.seatMaximum} members
          </li>
        )}
        {!plan.subscriptionRequired && (
          <li className="flex items-start gap-2">
            <Check className="size-3.5 shrink-0" aria-hidden />
            No subscription fee
          </li>
        )}
      </ul>
      {!subscriptionActive && (
        <Button
          size="sm"
          variant={current ? "outline" : "default"}
          disabled={busy || (current && !needsCard)}
          onClick={onSelect}
        >
          {planActionLabel(plan, current && !needsCard)}
        </Button>
      )}
      {needsCard && <p className="mt-2 text-xs text-muted-foreground">Add a card to activate this plan.</p>}
    </div>
  );
}

function planActionLabel(plan: OrganizationsBillingPlanOption, current: boolean) {
  if (current) return "Current plan";
  if (plan.perSeat) return `Review ${plan.name} seats`;
  return plan.subscriptionRequired ? `Subscribe to ${plan.name}` : `Select ${plan.name}`;
}
