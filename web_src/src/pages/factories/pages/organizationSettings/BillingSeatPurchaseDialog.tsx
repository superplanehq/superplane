import { useId } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { recentSeatPurchase } from "@/lib/seatPurchases";
import type { OrganizationsPreviewSeatPurchaseResponse } from "@/api-client";
import type { UseSeatPurchaseResult } from "@/hooks/useSeatPurchase";
import { billingPeriodEndLabel } from "../../lib/billingPlans";
import { formatUsdCents } from "../../lib/workOrderUsage";
import { BillingSeatPurchaseLimit, BillingSeatPurchaseStatus } from "./BillingSeatPurchaseStatus";

export type BillingSeatPurchaseDialogProps = {
  purchase: UseSeatPurchaseResult;
  purchasableSeats?: number;
  planMaximumSeats?: number;
  cancelAtPeriodEnd: boolean;
  paymentPending: boolean;
  paymentError?: string;
  onUpdatePaymentMethod: () => void;
};

export function BillingSeatPurchaseDialog({
  purchase,
  purchasableSeats,
  planMaximumSeats,
  cancelAtPeriodEnd,
  paymentPending,
  paymentError,
  onUpdatePaymentMethod,
}: BillingSeatPurchaseDialogProps) {
  const visiblePurchase = recentSeatPurchase(purchase.purchase);
  const showForm = canShowPurchaseForm({ purchase, purchasableSeats, cancelAtPeriodEnd });
  const blocked = purchase.submitting || paymentPending;

  return (
    <Dialog open={purchase.isOpen} onOpenChange={purchase.setOpen}>
      <DialogContent showCloseButton={false}>
        <DialogHeader>
          <DialogTitle>Buy seats</DialogTitle>
          <DialogDescription>Add seats to your current subscription.</DialogDescription>
        </DialogHeader>
        <PurchaseAvailability
          cancelAtPeriodEnd={cancelAtPeriodEnd}
          purchasableSeats={purchasableSeats}
          planMaximumSeats={planMaximumSeats}
        />
        {showForm && <SeatPurchaseForm purchase={purchase} maximum={purchasableSeats} disabled={blocked} />}
        {purchase.previewError && <p role="alert">{purchase.previewError}</p>}
        <BillingSeatPurchaseStatus
          purchase={visiblePurchase}
          message={purchase.purchaseMessage}
          timedOut={purchase.timedOut}
          refreshing={purchase.refreshing}
          paymentPending={paymentPending}
          submissionUnresolved={purchase.submissionUnresolved}
          onRefresh={purchase.refreshBilling}
          onUpdatePaymentMethod={onUpdatePaymentMethod}
        />
        {paymentPending && <p role="status">Update your card in the payment window, then close it.</p>}
        {paymentError && <p role="alert">{paymentError}</p>}
        <DialogFooter>
          <Button variant="outline" onClick={() => purchase.setOpen(false)}>
            Close
          </Button>
          {showForm && (
            <Button disabled={blocked || !purchase.canConfirm} onClick={() => void purchase.confirm()}>
              {purchase.submitting ? "Buying seats…" : "Confirm purchase"}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function canShowPurchaseForm({
  purchase,
  purchasableSeats,
  cancelAtPeriodEnd,
}: Pick<BillingSeatPurchaseDialogProps, "purchase" | "purchasableSeats" | "cancelAtPeriodEnd">) {
  if (purchasableSeats === 0 || cancelAtPeriodEnd) return false;
  if (["pending", "awaiting_confirmation"].includes(purchase.purchase?.status ?? "")) return false;
  return !purchase.attemptSucceeded;
}

function PurchaseAvailability({
  cancelAtPeriodEnd,
  purchasableSeats,
  planMaximumSeats,
}: Pick<BillingSeatPurchaseDialogProps, "cancelAtPeriodEnd" | "purchasableSeats" | "planMaximumSeats">) {
  if (cancelAtPeriodEnd) return <p role="status">Resume the subscription before buying seats.</p>;
  if (purchasableSeats === 0) return <BillingSeatPurchaseLimit maximum={planMaximumSeats} />;
  return null;
}

function SeatPurchaseForm({
  purchase,
  maximum,
  disabled,
}: {
  purchase: UseSeatPurchaseResult;
  maximum?: number;
  disabled: boolean;
}) {
  return (
    <div className="space-y-4">
      <SeatPurchaseQuantity
        value={purchase.additionalSeats}
        maximum={maximum}
        disabled={disabled}
        onChange={purchase.setAdditionalSeats}
      />
      {purchase.previewPending && <p role="status">Loading estimate…</p>}
      {purchase.preview && <SeatPurchaseEstimate preview={purchase.preview} />}
    </div>
  );
}

function validSeatQuantity(value: string, maximum: number | undefined) {
  const quantity = Number(value);
  return /^\d+$/.test(value) && Number.isSafeInteger(quantity) && quantity >= 1 && quantity <= (maximum ?? Infinity);
}

function SeatPurchaseQuantity({
  value,
  maximum,
  disabled,
  onChange,
}: {
  value: string;
  maximum?: number;
  disabled: boolean;
  onChange: (value: string) => void;
}) {
  const inputId = useId();
  const quantity = Number(value);
  const cannotStep = disabled || !Number.isInteger(quantity);
  return (
    <>
      <Label htmlFor={inputId}>Additional seats</Label>
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          aria-label="Decrease additional seats"
          disabled={cannotStep || quantity <= 1}
          onClick={() => onChange(String(quantity - 1))}
        >
          −
        </Button>
        <Input
          id={inputId}
          type="number"
          min={1}
          max={maximum}
          step={1}
          value={value}
          disabled={disabled}
          aria-invalid={!validSeatQuantity(value, maximum)}
          onChange={(event) => onChange(event.target.value)}
          className="w-24"
        />
        <Button
          variant="outline"
          aria-label="Increase additional seats"
          disabled={cannotStep || quantity >= (maximum ?? Infinity)}
          onClick={() => onChange(String(quantity + 1))}
        >
          +
        </Button>
      </div>
    </>
  );
}

function SeatPurchaseEstimate({ preview }: { preview: OrganizationsPreviewSeatPurchaseResponse }) {
  return (
    <div className="space-y-2 text-sm">
      <p>Estimated charge now: {formatUsdCents(Number(preview.proratedChargeCents ?? 0))}</p>
      <p>today to {billingPeriodEndLabel(preview.periodEnd) ?? "renewal date unavailable"}</p>
      <p>New monthly total: {formatUsdCents(Number(preview.newMonthlyTotalCents ?? 0))}</p>
      <p className="text-muted-foreground">Estimate. Tax is calculated at payment.</p>
    </div>
  );
}
