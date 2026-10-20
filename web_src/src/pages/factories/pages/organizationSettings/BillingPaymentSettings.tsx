import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { organizationsCreateBillingPortalSession, organizationsSelectBillingPlan } from "@/api-client";
import { Button } from "@/components/ui/button";
import { organizationBillingQueryKey, useOrganizationBilling } from "@/hooks/useOrganizationBilling";
import { getApiErrorMessage } from "@/lib/errors";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { BillingCardSetup } from "./BillingCardSetup";
import { FactorySettingsCard } from "../settings/FactorySettingsCard";
import { formatUsdCents } from "../../lib/workOrderUsage";

export function BillingPaymentSettings({
  organizationId,
  canManageBilling,
}: {
  organizationId: string;
  canManageBilling: boolean;
}) {
  const { data: billing } = useOrganizationBilling(organizationId);
  const queryClient = useQueryClient();
  const [error, setError] = useState("");
  const action = useMutation({
    mutationFn: async (planKey?: string) => {
      setError("");
      if (!planKey) {
        const response = await organizationsCreateBillingPortalSession(
          withOrganizationHeader({ organizationId, path: { id: organizationId }, body: {} }),
        );
        if (!response.data?.portalUrl) throw new Error("Could not open payment settings.");
        window.location.assign(response.data.portalUrl);
        return;
      }
      const response = await organizationsSelectBillingPlan(
        withOrganizationHeader({ organizationId, path: { id: organizationId }, body: { planKey } }),
      );
      if (response.data?.checkoutUrl) {
        window.location.assign(response.data.checkoutUrl);
        return;
      }
      await queryClient.invalidateQueries({ queryKey: organizationBillingQueryKey(organizationId) });
    },
    onError: (cause) => setError(getApiErrorMessage(cause, "Could not update billing.")),
  });

  if (!billing?.billingEnabled) return null;
  const plans = billing.availablePlans ?? [];
  const subscriptionActive = billing.planSource === "polar" && billing.creditPurchaseAllowed;
  const manageInPortal = subscriptionActive || billing.hasPaymentMethod;
  return (
    <FactorySettingsCard title="Payment method">
      <p>{billing.hasPaymentMethod ? "A saved card is available." : "No saved card is available."}</p>
      {billing.requirePaymentMethod && !billing.hasPaymentMethod && (
        <p role="status">Your plan requires a saved card before usage.</p>
      )}
      {canManageBilling ? (
        <div className="mt-3 space-y-3">
          {!manageInPortal && <BillingCardSetup organizationId={organizationId} disabled={action.isPending} />}
          <div className="flex flex-wrap gap-2">
            {manageInPortal && (
              <Button disabled={action.isPending} onClick={() => action.mutate(undefined)}>
                {subscriptionActive ? "Manage subscription and card" : "Manage payment method"}
              </Button>
            )}
          </div>
          {plans.map((plan) => {
            const current = billing.catalogKey === plan.key;
            const needsCard = plan.requirePaymentMethod && !billing.hasPaymentMethod && !plan.subscriptionRequired;
            return (
              <div key={plan.key} className="flex flex-wrap items-center justify-between gap-3 rounded border p-3">
                <div>
                  <p className="font-medium">{plan.name}</p>
                  <p className="text-sm text-muted-foreground">
                    {formatUsdCents(Number(plan.basePriceCents ?? 0))} base price
                    {Number(plan.seatPriceCents ?? 0) > 0 &&
                      ` + ${formatUsdCents(Number(plan.seatPriceCents))} per seat`}
                    {plan.requirePaymentMethod && " · Card required"}
                  </p>
                  {needsCard && <p className="text-sm">Add a payment method before selecting this plan.</p>}
                </div>
                <Button
                  variant="outline"
                  disabled={current || needsCard || action.isPending || subscriptionActive}
                  onClick={() => action.mutate(plan.key)}
                >
                  {current
                    ? "Current plan"
                    : plan.subscriptionRequired
                      ? `Subscribe to ${plan.name}`
                      : `Select ${plan.name}`}
                </Button>
              </div>
            );
          })}
          {subscriptionActive && (
            <p className="text-sm text-muted-foreground">
              Manage your existing subscription in the billing portal to change plans.
            </p>
          )}
          {error && (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          )}
        </div>
      ) : (
        <p className="mt-2 text-sm text-muted-foreground">
          Contact an organization owner to set up payment or change plans.
        </p>
      )}
    </FactorySettingsCard>
  );
}
