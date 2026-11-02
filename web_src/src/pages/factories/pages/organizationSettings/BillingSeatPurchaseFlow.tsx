import { useEffect, useRef, useState, type ReactNode } from "react";
import { useSearchParams } from "react-router";
import type { OrganizationsDescribeOrganizationBillingResponse } from "@/api-client";
import { useBillingCardSetup } from "@/hooks/useBillingCardSetup";
import { useOrganizationBilling } from "@/hooks/useOrganizationBilling";
import { useSeatPurchase, type UseSeatPurchaseResult } from "@/hooks/useSeatPurchase";
import { recentSeatPurchase } from "@/lib/seatPurchases";
import { BillingSeatPurchaseDialog } from "./BillingSeatPurchaseDialog";
import { BillingSeatPurchaseStatus } from "./BillingSeatPurchaseStatus";
import { BillingSeatsCard } from "./BillingSeatsCard";
import { FactorySettingsCard } from "../settings/FactorySettingsCard";
import type { BillingSeatSummary } from "./useOrganizationBillingPageModel";

type BillingSeatPurchaseFlowProps = {
  organizationId: string;
  canManageBilling: boolean;
  seats?: BillingSeatSummary;
  currentPeriodEnd?: string;
  cancelAtPeriodEnd: boolean;
  loading: boolean;
  children: (seatsCard: ReactNode, dialog: ReactNode) => ReactNode;
};

export function BillingSeatPurchaseFlow(props: BillingSeatPurchaseFlowProps) {
  const { organizationId, canManageBilling, seats, currentPeriodEnd, cancelAtPeriodEnd } = props;
  const { data: billing } = useOrganizationBilling(organizationId);
  const canBuySeats = seatPurchasesAllowed(billing, canManageBilling);
  const purchase = useSeatPurchase({
    organizationId,
    billing,
    enabled: canBuySeats,
    pollingEnabled: canManageBilling,
  });
  const cardSetup = useBillingCardSetup(organizationId);
  const canRecoverPurchase = canManageBilling;
  const updatePaymentMethod = () => cardSetup.open(purchase.requestPreview);
  const denied = useSeatPurchaseDeepLink({
    organizationId,
    canBuySeats,
    canManageBilling,
    loading: props.loading,
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
        />
      )}
      <BillingSeatPurchaseFeedback
        purchase={purchase}
        cardSetup={cardSetup}
        canRecoverPurchase={canRecoverPurchase}
        onUpdatePaymentMethod={updatePaymentMethod}
      />
    </>
  );
  const dialog =
    canRecoverPurchase && (canBuySeats || purchase.isOpen) ? (
      <BillingSeatPurchaseDialog
        purchase={purchase}
        purchasableSeats={billing?.purchasableSeats}
        planMaximumSeats={billing?.planMaximumSeats}
        cancelAtPeriodEnd={cancelAtPeriodEnd}
        purchasesAllowed={canBuySeats}
        paymentPending={cardSetup.isPending}
        paymentError={cardSetup.error}
        onUpdatePaymentMethod={updatePaymentMethod}
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
  const consumedFor = useRef<string | undefined>(undefined);
  useEffect(
    () => () => {
      consumedFor.current = undefined;
    },
    [],
  );
  useEffect(() => {
    if (searchParams.get("buySeats") !== "1") {
      consumedFor.current = undefined;
      return;
    }
    if (loading || consumedFor.current === organizationId) return;
    // Router navigation can stay pending after the dialog opens or closes.
    consumedFor.current = organizationId;
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
  canRecoverPurchase,
  onUpdatePaymentMethod,
}: {
  purchase: UseSeatPurchaseResult;
  cardSetup: ReturnType<typeof useBillingCardSetup>;
  canRecoverPurchase: boolean;
  onUpdatePaymentMethod: () => void;
}) {
  const visiblePurchase = recentSeatPurchase(purchase.purchase);
  if (!visiblePurchase && !purchase.submissionUnresolved) return null;
  return (
    <FactorySettingsCard title="Seat purchase" data-testid="billing-seat-purchase">
      <BillingSeatPurchaseStatus
        purchase={visiblePurchase}
        message={purchase.submissionUnresolved ? purchase.purchaseMessage : undefined}
        timedOut={canRecoverPurchase && purchase.timedOut}
        submissionUnresolved={canRecoverPurchase && purchase.submissionUnresolved}
        refreshing={purchase.refreshing}
        paymentPending={cardSetup.isPending}
        onRefresh={purchase.refreshBilling}
        onUpdatePaymentMethod={canRecoverPurchase ? onUpdatePaymentMethod : undefined}
      />
      {!purchase.isOpen && cardSetup.isPending && (
        <p role="status">Update your card in the payment window, then close it.</p>
      )}
      {!purchase.isOpen && cardSetup.error && <p role="alert">{cardSetup.error}</p>}
    </FactorySettingsCard>
  );
}
