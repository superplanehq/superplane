import { useRef, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  organizationsCreateBillingPortalSession,
  organizationsCreatePaymentMethodSession,
  organizationsSelectBillingPlan,
} from "@/api-client";
import { polarPortalSettingsUrl, useBillingCardSetup } from "@/hooks/useBillingCardSetup";
import { organizationBillingQueryKey, useOrganizationBilling } from "@/hooks/useOrganizationBilling";
import { getApiErrorMessage } from "@/lib/errors";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { BillingPaymentFailedBanner } from "./BillingPaymentFailedBanner";
import { BillingCurrentPlan } from "./BillingCurrentPlan";
import { BillingChangePlanDialog } from "./BillingChangePlanDialog";
import { billingIsPastDue, billingPaymentBlocked } from "../../lib/billingPlans";

type BillingAction = { planKey?: string; seats?: number; paymentMethod?: boolean };

type BillingPaymentSettingsProps = {
  organizationId: string;
  canManageBilling: boolean;
  subscriptionPending?: boolean;
  onCancelSubscription?: () => void | Promise<void>;
  onKeepSubscription?: () => void | Promise<void>;
};

export function BillingPaymentSettings({
  organizationId,
  canManageBilling,
  subscriptionPending = false,
  onCancelSubscription,
  onKeepSubscription,
}: BillingPaymentSettingsProps) {
  const { data: billing } = useOrganizationBilling(organizationId);
  const changePlanButton = useRef<HTMLButtonElement>(null);
  const [plansOpen, setPlansOpen] = useState(false);
  const cardSetup = useBillingCardSetup(organizationId);
  const payment = useBillingPaymentAction(organizationId, () => setPlansOpen(false));

  if (!billing?.billingEnabled) return null;
  const busy = payment.pending || cardSetup.isPending || subscriptionPending;
  const paymentBlocked = billingPaymentBlocked(billing.polarSubscriptionStatus);
  const subscriptionActive = billing.planSource === "polar" && billing.creditPurchaseAllowed;
  const manageInPortal = subscriptionActive || billing.hasPaymentMethod || paymentBlocked;
  const feedback = <BillingPaymentFeedback setup={cardSetup} error={payment.error} />;
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
      <BillingCurrentPlan
        changePlanButtonRef={changePlanButton}
        billing={billing}
        canManageBilling={canManageBilling}
        busy={busy}
        onChangePlan={() => setPlansOpen(true)}
        onManagePayment={() => (manageInPortal ? payment.submit() : cardSetup.open())}
        onCancel={onCancelSubscription}
        onKeep={onKeepSubscription}
      />
      {!plansOpen && feedback}
      {canManageBilling && !paymentBlocked && (
        <BillingChangePlanDialog
          onCloseAutoFocus={() => changePlanButton.current?.focus()}
          open={plansOpen}
          onOpenChange={(open) => {
            if (!busy) setPlansOpen(open);
          }}
          billing={billing}
          busy={busy}
          subscriptionActive={Boolean(subscriptionActive)}
          onSelect={(plan) => {
            if (plan.requirePaymentMethod && !billing.hasPaymentMethod && !plan.subscriptionRequired) {
              cardSetup.open(() => payment.submit(plan.key));
              return;
            }
            payment.submit(plan.key);
          }}
          onSubscribeWithSeats={payment.submit}
          onManageSubscription={() => payment.submit()}
          feedback={feedback}
        />
      )}
    </>
  );
}

function useBillingPaymentAction(organizationId: string, onComplete: () => void) {
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
      onComplete();
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

function BillingPaymentFeedback({ setup, error }: { setup: ReturnType<typeof useBillingCardSetup>; error: string }) {
  const message = error || setup.error;
  return (
    <>
      {setup.isPending && (
        <p role="status" className="text-sm text-muted-foreground">
          {setup.opening ? "Opening card setup…" : "Waiting for your saved card…"}
        </p>
      )}
      {message && (
        <p role="alert" className="text-sm text-destructive">
          {message}
        </p>
      )}
    </>
  );
}
