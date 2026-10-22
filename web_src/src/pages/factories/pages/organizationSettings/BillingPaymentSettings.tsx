import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { organizationsCreateBillingPortalSession, organizationsSelectBillingPlan } from "@/api-client";
import { Button } from "@/components/ui/button";
import { useBillingCardSetup } from "@/hooks/useBillingCardSetup";
import { organizationBillingQueryKey, useOrganizationBilling } from "@/hooks/useOrganizationBilling";
import { getApiErrorMessage } from "@/lib/errors";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
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
  const cardSetup = useBillingCardSetup(organizationId);
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
  const busy = action.isPending || cardSetup.isPending;
  const cardRequiredPlan = plans.some((plan) => plan.requirePaymentMethod && !plan.subscriptionRequired);
  return (
    <FactorySettingsCard title="Payment method">
      <p>{billing.hasPaymentMethod ? "A saved card is available." : "No saved card is available."}</p>
      {billing.requirePaymentMethod && !billing.hasPaymentMethod && (
        <p role="status">Your plan requires a saved card before usage.</p>
      )}
      {canManageBilling ? (
        <div className="mt-3 space-y-3">
          {!manageInPortal && !cardRequiredPlan && (
            <Button disabled={busy} onClick={() => cardSetup.open()}>
              Add payment method
            </Button>
          )}
          <div className="flex flex-wrap gap-2">
            {manageInPortal && (
              <Button disabled={busy} onClick={() => action.mutate(undefined)}>
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
                    {formatUsdCents(Number(plan.priceCents ?? 0))}
                    {plan.perSeat && " per seat"}
                    {plan.requirePaymentMethod && " · Card required"}
                  </p>
                  {needsCard && (
                    <p className="text-sm text-muted-foreground">
                      Save a card in the Polar popup to activate this plan. No subscription required.
                    </p>
                  )}
                </div>
                <Button
                  variant="outline"
                  disabled={(current && !needsCard) || busy || subscriptionActive}
                  onClick={() => {
                    setError("");
                    if (needsCard) {
                      cardSetup.open(() => action.mutate(plan.key));
                      return;
                    }
                    action.mutate(plan.key);
                  }}
                >
                  {current && !needsCard
                    ? "Current plan"
                    : plan.subscriptionRequired
                      ? `Subscribe to ${plan.name}`
                      : `Select ${plan.name}`}
                </Button>
              </div>
            );
          })}
          <BillingPaymentFeedback setup={cardSetup} error={error} subscriptionActive={Boolean(subscriptionActive)} />
        </div>
      ) : (
        <p className="mt-2 text-sm text-muted-foreground">
          Contact an organization owner to set up payment or change plans.
        </p>
      )}
    </FactorySettingsCard>
  );
}

function BillingPaymentFeedback({
  setup,
  error,
  subscriptionActive,
}: {
  setup: ReturnType<typeof useBillingCardSetup>;
  error: string;
  subscriptionActive: boolean;
}) {
  const message = error || setup.error;
  return (
    <>
      {subscriptionActive && (
        <p className="text-sm text-muted-foreground">
          Manage your existing subscription in the billing portal to change plans.
        </p>
      )}
      {setup.isPending && <p role="status">{setup.opening ? "Opening card setup…" : "Waiting for your saved card…"}</p>}
      {message && (
        <p role="alert" className="text-sm text-destructive">
          {message}
        </p>
      )}
    </>
  );
}
