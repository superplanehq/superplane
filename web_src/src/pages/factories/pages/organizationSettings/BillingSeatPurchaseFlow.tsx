import { useEffect, useState, type ReactNode } from "react";
import { useSearchParams } from "react-router";
import type { OrganizationsDescribeOrganizationBillingResponse } from "@/api-client";
import { useBillingCardSetup } from "@/hooks/useBillingCardSetup";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useOrganizationBilling } from "@/hooks/useOrganizationBilling";
import { useSeatPurchase, type UseSeatPurchaseResult } from "@/hooks/useSeatPurchase";
import { recentSeatPurchase } from "@/lib/seatPurchases";
import { BillingSeatPurchaseDialog } from "./BillingSeatPurchaseDialog";
import { BillingSeatPurchaseStatus } from "./BillingSeatPurchaseStatus";
import { BillingSeatsCard } from "./BillingSeatsCard";
import type { BillingSeatSummary } from "./useOrganizationBillingPageModel";

type BillingSeatPurchaseFlowProps = {
  organizationId: string;
  newBilling: boolean;
  canManageBilling: boolean;
  seats?: BillingSeatSummary;
  currentPeriodEnd?: string;
  cancelAtPeriodEnd: boolean;
  loading: boolean;
  children: (seatsCard: ReactNode, dialog: ReactNode) => ReactNode;
};

export function BillingSeatPurchaseFlow(props: BillingSeatPurchaseFlowProps) {
  const { organizationId, newBilling, canManageBilling, seats, currentPeriodEnd, cancelAtPeriodEnd } = props;
  const { data: billing } = useOrganizationBilling(organizationId);
  const features = useExperimentalFeature(organizationId);
  const canBuySeats = seatPurchasesAllowed(billing, newBilling && canManageBilling);
  const purchase = useSeatPurchase({
    organizationId,
    billing,
    enabled: canBuySeats,
    pollingEnabled: newBilling && canManageBilling,
  });
  const cardSetup = useBillingCardSetup(organizationId);
  const denied = useSeatPurchaseDeepLink({
    organizationId,
    canBuySeats,
    canManageBilling,
    loading: props.loading || features.isLoading || !features.organizationReady,
    setOpen: purchase.setOpen,
  });

  const seatsCard = (
    <>
      {denied && !seats && <p role="status">Ask an organization admin to buy seats.</p>}
      {seats && (
        <BillingSeatsCard
          seats={seats}
          currentPeriodEnd={currentPeriodEnd}
          cancelAtPeriodEnd={cancelAtPeriodEnd}
          canManageBilling={canManageBilling}
          canBuySeats={canBuySeats}
          onBuySeats={() => purchase.setOpen(true)}
          purchaseFeedback={
            <BillingSeatPurchaseFeedback
              purchase={purchase}
              cardSetup={cardSetup}
              canUpdatePaymentMethod={newBilling && canManageBilling}
            />
          }
        />
      )}
    </>
  );
  const dialog = canBuySeats ? (
    <BillingSeatPurchaseDialog
      purchase={purchase}
      purchasableSeats={billing?.purchasableSeats}
      planMaximumSeats={billing?.planMaximumSeats}
      cancelAtPeriodEnd={cancelAtPeriodEnd}
      paymentPending={cardSetup.isPending}
      paymentError={cardSetup.error}
      onUpdatePaymentMethod={() => cardSetup.open()}
    />
  ) : null;
  return props.children(seatsCard, dialog);
}

function seatPurchasesAllowed(
  billing: OrganizationsDescribeOrganizationBillingResponse | undefined,
  canManage: boolean,
) {
  return Boolean(
    canManage &&
      billing?.billingEnabled &&
      billing.planPerSeat &&
      billing.planSource === "polar" &&
      billing.polarSubscriptionStatus === "active" &&
      billing.creditPurchaseAllowed,
  );
}

function useSeatPurchaseDeepLink({
  organizationId,
  canBuySeats,
  canManageBilling,
  loading,
  setOpen,
}: {
  organizationId: string;
  canBuySeats: boolean;
  canManageBilling: boolean;
  loading: boolean;
  setOpen: (open: boolean) => void;
}) {
  const [searchParams, setSearchParams] = useSearchParams();
  const [deniedFor, setDeniedFor] = useState<string>();
  useEffect(() => {
    if (loading || searchParams.get("buySeats") !== "1") return;
    if (!canManageBilling) setDeniedFor(organizationId);
    if (canBuySeats) setOpen(true);
    setSearchParams(
      (current) => {
        const next = new URLSearchParams(current);
        next.delete("buySeats");
        return next;
      },
      { replace: true },
    );
  }, [organizationId, canBuySeats, canManageBilling, loading, setOpen, searchParams, setSearchParams]);
  return deniedFor === organizationId && !canManageBilling;
}

function BillingSeatPurchaseFeedback({
  purchase,
  cardSetup,
  canUpdatePaymentMethod,
}: {
  purchase: UseSeatPurchaseResult;
  cardSetup: ReturnType<typeof useBillingCardSetup>;
  canUpdatePaymentMethod: boolean;
}) {
  const visiblePurchase = recentSeatPurchase(purchase.purchase);
  if (!visiblePurchase) return null;
  return (
    <>
      <BillingSeatPurchaseStatus
        purchase={visiblePurchase}
        timedOut={purchase.timedOut}
        refreshing={purchase.refreshing}
        paymentPending={cardSetup.isPending}
        onRefresh={purchase.refreshBilling}
        onUpdatePaymentMethod={canUpdatePaymentMethod ? () => cardSetup.open() : undefined}
      />
      {!purchase.isOpen && cardSetup.isPending && (
        <p role="status">Update your card in the payment window, then close it.</p>
      )}
      {!purchase.isOpen && cardSetup.error && <p role="alert">{cardSetup.error}</p>}
    </>
  );
}
