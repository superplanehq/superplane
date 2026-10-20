import { SUPERPLANE_PRICING_URL } from "@/lib/pricing";
import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { FactorySettingsCard } from "../settings/FactorySettingsCard";
import { BillingCancelBusinessDialog } from "./BillingCancelBusinessDialog";
import { billingCanCancelBusiness, billingCanKeepBusiness, billingPeriodEndLabel } from "../../lib/billingPlans";

export function BillingPlansSection({
  canManageBilling,
  creditPurchaseAllowed,
  planSource,
  catalogName,
  plan,
  cancelAtPeriodEnd,
  currentPeriodEnd,
  pending,
  onCancel,
  onKeep,
}: {
  canManageBilling: boolean;
  creditPurchaseAllowed: boolean;
  planSource?: string;
  catalogName?: string;
  plan?: string;
  cancelAtPeriodEnd?: boolean;
  currentPeriodEnd?: string;
  pending: boolean;
  onCancel?: () => void | Promise<void>;
  onKeep?: () => void | Promise<void>;
}) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const canCancel = billingCanCancelBusiness({
    canManageBilling,
    creditPurchaseAllowed,
    planSource,
    cancelAtPeriodEnd,
  });
  const canKeep = billingCanKeepBusiness({ canManageBilling, creditPurchaseAllowed, planSource, cancelAtPeriodEnd });
  const end = billingPeriodEndLabel(currentPeriodEnd);
  return (
    <FactorySettingsCard title="Plans" data-testid="billing-plans">
      <div data-testid={plan === "business" ? "billing-plan-business" : "billing-plan-current"}>
        <h3 className="text-sm font-medium">
          {catalogName || (plan === "business" ? "Business" : plan === "trial" ? "Trial" : "No plan")}
        </h3>
        {creditPurchaseAllowed && (
          <Badge variant="outline" data-testid="billing-current-plan">
            Current plan
          </Badge>
        )}
        {plan === "business" && (
          <p className="mt-2">
            <span className="text-xl font-semibold">$199</span> per month
          </p>
        )}
        {cancelAtPeriodEnd && end && (
          <p data-testid="billing-subscription-ends">
            Subscription ends on {end}. Hosted runs continue until that date.
          </p>
        )}
        {canCancel && onCancel && (
          <Button
            variant="outline"
            disabled={pending}
            data-testid="billing-cancel-subscription"
            onClick={() => setConfirmOpen(true)}
          >
            Cancel subscription
          </Button>
        )}
        {canKeep && onKeep && (
          <Button disabled={pending} data-testid="billing-keep-subscription" onClick={() => void onKeep()}>
            Keep subscription
          </Button>
        )}
      </div>
      <p className="mt-3 text-sm text-muted-foreground">
        New subscriptions are unavailable. Contact an installation admin to change your plan.
      </p>
      <p className="mt-3">
        <a href={SUPERPLANE_PRICING_URL} target="_blank" rel="noreferrer">
          Talk to us
        </a>
      </p>
      <BillingCancelBusinessDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        currentPeriodEnd={currentPeriodEnd}
        pending={pending}
        onConfirm={async () => {
          await onCancel?.();
          setConfirmOpen(false);
        }}
      />
    </FactorySettingsCard>
  );
}
