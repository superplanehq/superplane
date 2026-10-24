import { useRef, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  organizationsCreateBillingPortalSession,
  organizationsCreatePaymentMethodSession,
  organizationsSelectBillingPlan,
  type OrganizationsBillingPlanOption,
  type OrganizationsDescribeOrganizationBillingResponse,
} from "@/api-client";
import { Button } from "@/components/ui/button";
import { polarPortalSettingsUrl, useBillingCardSetup } from "@/hooks/useBillingCardSetup";
import { organizationBillingQueryKey, useOrganizationBilling } from "@/hooks/useOrganizationBilling";
import { getApiErrorMessage } from "@/lib/errors";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { BillingPaymentFailedBanner } from "./BillingPaymentFailedBanner";
import { BillingSeatCheckout } from "./BillingSeatCheckout";
import { FactorySettingsCard } from "../settings/FactorySettingsCard";
import { billingIsPastDue, billingPaymentBlocked } from "../../lib/billingPlans";
import { formatUsdCents } from "../../lib/workOrderUsage";

type BillingAction = { planKey?: string; seats?: number; paymentMethod?: boolean };

export function BillingPaymentSettings({
  organizationId,
  canManageBilling,
}: {
  organizationId: string;
  canManageBilling: boolean;
}) {
  const { data: billing } = useOrganizationBilling(organizationId);
  const cardSetup = useBillingCardSetup(organizationId);
  const payment = useBillingPaymentAction(organizationId);

  if (!billing?.billingEnabled) return null;
  const busy = payment.pending || cardSetup.isPending;
  return (
    <>
      {billingIsPastDue(billing.polarSubscriptionStatus) && (
        <BillingPaymentFailedBanner
          failedAt={billing.paymentFailedAt}
          accessEndsAt={billing.paidAccessEndsAt}
          canManageBilling={canManageBilling}
          pending={busy}
          onUpdatePaymentMethod={payment.updatePaymentMethod}
        />
      )}
      <FactorySettingsCard title="Payment method">
        <p>{billing.hasPaymentMethod ? "A saved card is available." : "No saved card is available."}</p>
        {billing.requirePaymentMethod && !billing.hasPaymentMethod && (
          <p role="status">Your plan requires a saved card before usage.</p>
        )}
        {canManageBilling ? (
          <BillingPaymentControls
            billing={billing}
            busy={busy}
            cardSetup={cardSetup}
            error={payment.error}
            onSelect={payment.submit}
          />
        ) : (
          <p className="mt-2 text-sm text-muted-foreground">
            Ask an organization admin to set up payment or change plans.
          </p>
        )}
      </FactorySettingsCard>
    </>
  );
}

function useBillingPaymentAction(organizationId: string) {
  const { refetch } = useOrganizationBilling(organizationId);
  const queryClient = useQueryClient();
  const [error, setError] = useState("");
  const submitting = useRef(false);
  const action = useMutation({
    retry: false,
    mutationFn: async ({ planKey, seats, paymentMethod }: BillingAction) => {
      setError("");
      if (paymentMethod) {
        const response = await organizationsCreatePaymentMethodSession(
          withOrganizationHeader({ organizationId, path: { id: organizationId }, body: {} }),
        );
        if (!response.data?.portalUrl) throw new Error("Could not open payment settings.");
        window.location.assign(polarPortalSettingsUrl(response.data.portalUrl));
        return;
      }
      if (!planKey) {
        const response = await organizationsCreateBillingPortalSession(
          withOrganizationHeader({ organizationId, path: { id: organizationId }, body: {} }),
        );
        if (!response.data?.portalUrl) throw new Error("Could not open payment settings.");
        window.location.assign(response.data.portalUrl);
        return;
      }
      const response = await organizationsSelectBillingPlan(
        withOrganizationHeader({
          organizationId,
          path: { id: organizationId },
          body: { planKey, ...(seats == null ? {} : { seats }) },
        }),
      );
      if (response.data?.checkoutUrl) {
        window.location.assign(response.data.checkoutUrl);
        return;
      }
      await queryClient.invalidateQueries({ queryKey: organizationBillingQueryKey(organizationId) });
    },
    onError: async (cause) => {
      setError(getApiErrorMessage(cause, "Could not update billing."));
      await refetch().catch(() => undefined);
    },
    onSettled: () => {
      submitting.current = false;
    },
  });

  const run = (next: BillingAction) => {
    if (submitting.current) return;
    submitting.current = true;
    action.mutate(next);
  };

  return {
    error,
    pending: action.isPending,
    submit: (planKey?: string, seats?: number) => run({ planKey, seats }),
    updatePaymentMethod: () => run({ paymentMethod: true }),
  };
}

function BillingPaymentControls({
  billing,
  busy,
  cardSetup,
  error,
  onSelect,
}: {
  billing: OrganizationsDescribeOrganizationBillingResponse;
  busy: boolean;
  cardSetup: ReturnType<typeof useBillingCardSetup>;
  error: string;
  onSelect: (planKey?: string, seats?: number) => void;
}) {
  const pastDue = billingIsPastDue(billing.polarSubscriptionStatus);
  // A failed or paused payment locks plan changes on the server. Hide the checkout it would reject.
  const paymentBlocked = billingPaymentBlocked(billing.polarSubscriptionStatus);
  const plans = paymentBlocked ? [] : (billing.availablePlans ?? []);
  const subscriptionActive = billing.planSource === "polar" && billing.creditPurchaseAllowed;
  const manageInPortal = subscriptionActive || billing.hasPaymentMethod || paymentBlocked;
  const cardRequiredPlan = plans.some((plan) => plan.requirePaymentMethod && !plan.subscriptionRequired);
  return (
    <div className="mt-3 space-y-3">
      {!paymentBlocked && !manageInPortal && !cardRequiredPlan && (
        <Button disabled={busy} onClick={() => cardSetup.open()}>
          Add payment method
        </Button>
      )}
      <div className="flex flex-wrap gap-2">
        {manageInPortal && !pastDue && (
          <Button disabled={busy} onClick={() => onSelect()}>
            {subscriptionActive ? "Manage subscription and card" : "Manage payment method"}
          </Button>
        )}
      </div>
      {plans.map((plan) => (
        <BillingPlanChoice
          key={plan.key}
          plan={plan}
          billing={billing}
          busy={busy}
          subscriptionActive={Boolean(subscriptionActive)}
          cardSetup={cardSetup}
          onSelect={onSelect}
        />
      ))}
      <BillingPaymentFeedback
        setup={cardSetup}
        error={error}
        subscriptionActive={Boolean(subscriptionActive) && !paymentBlocked}
      />
    </div>
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

function BillingPlanChoice({
  plan,
  billing,
  busy,
  subscriptionActive,
  cardSetup,
  onSelect,
}: {
  plan: OrganizationsBillingPlanOption;
  billing: OrganizationsDescribeOrganizationBillingResponse;
  busy: boolean;
  subscriptionActive: boolean;
  cardSetup: ReturnType<typeof useBillingCardSetup>;
  onSelect: (planKey: string | undefined, seats?: number) => void;
}) {
  const current = billing.catalogKey === plan.key;
  const needsCard = Boolean(plan.requirePaymentMethod && !billing.hasPaymentMethod && !plan.subscriptionRequired);
  const canStartSeatCheckout = plan.perSeat && !current && !subscriptionActive;
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded border p-3">
      <div>
        <p className="font-medium">{plan.name}</p>
        <p className="text-sm text-muted-foreground">{billingPlanPrice(plan)}</p>
        {needsCard && (
          <p className="text-sm text-muted-foreground">
            Save a card in the Polar popup to activate this plan. No subscription required.
          </p>
        )}
      </div>
      {canStartSeatCheckout ? (
        <BillingSeatCheckout
          plan={plan}
          occupied={billing.occupiedSeats}
          disabled={busy}
          onSubscribe={(seats) => onSelect(plan.key, seats)}
        />
      ) : (
        <BillingPlanButton
          plan={plan}
          current={current && !needsCard}
          needsCard={needsCard}
          disabled={busy || subscriptionActive}
          onSelect={() => onSelect(plan.key)}
          onSaveCard={(select) => cardSetup.open(select)}
        />
      )}
    </div>
  );
}

function billingPlanPrice(plan: OrganizationsBillingPlanOption): string {
  const period = plan.perSeat ? " per seat per month" : plan.subscriptionRequired ? " per month" : "";
  const card = plan.requirePaymentMethod ? " · Card required" : "";
  return `${formatUsdCents(Number(plan.priceCents ?? 0))}${period}${card}`;
}

function BillingPlanButton({
  plan,
  current,
  needsCard,
  disabled,
  onSelect,
  onSaveCard,
}: {
  plan: OrganizationsBillingPlanOption;
  current: boolean;
  needsCard: boolean;
  disabled: boolean;
  onSelect: () => void;
  onSaveCard: (select: () => void) => void;
}) {
  return (
    <Button
      variant="outline"
      disabled={current || disabled}
      onClick={() => {
        if (needsCard) {
          onSaveCard(onSelect);
          return;
        }
        onSelect();
      }}
    >
      {current ? "Current plan" : plan.subscriptionRequired ? `Subscribe to ${plan.name}` : `Select ${plan.name}`}
    </Button>
  );
}
