import type { OrganizationsSeatPurchase } from "@/api-client";
import { Button } from "@/components/ui/button";

export function BillingSeatPurchaseLimit({ maximum }: { maximum?: number }) {
  return (
    <p role="status" className="text-sm text-muted-foreground">
      Your plan allows {maximum ?? "no additional"} seats. Remove a member or cancel an invitation to free a seat.
    </p>
  );
}

export function BillingSeatPurchaseStatus({
  purchase,
  message,
  timedOut,
  refreshing,
  paymentPending,
  onRefresh,
  onUpdatePaymentMethod,
}: {
  purchase?: OrganizationsSeatPurchase;
  message?: string | null;
  timedOut: boolean;
  refreshing: boolean;
  paymentPending: boolean;
  onRefresh: () => void | Promise<void>;
  onUpdatePaymentMethod?: () => void;
}) {
  const paymentFailed = ["failed", "action_required"].includes(purchase?.status ?? "");
  const statusMessage = seatPurchaseStatusMessage(purchase, message);
  if (!statusMessage && !timedOut) return null;

  return (
    <div className="space-y-3" aria-live="polite">
      {statusMessage && <p role={paymentFailed ? "alert" : "status"}>{statusMessage}</p>}
      {timedOut && (
        <>
          <p role="status">Polar has not confirmed the payment yet. Seats are added after Polar confirms it.</p>
          <Button variant="outline" disabled={refreshing} onClick={() => void onRefresh()}>
            {refreshing ? "Refreshing billing…" : "Refresh billing"}
          </Button>
        </>
      )}
      {paymentFailed && onUpdatePaymentMethod && (
        <Button variant="outline" disabled={paymentPending} onClick={onUpdatePaymentMethod}>
          Update payment method
        </Button>
      )}
    </div>
  );
}

function seatPurchaseStatusMessage(
  purchase: OrganizationsSeatPurchase | undefined,
  message: string | null | undefined,
) {
  if (message) return message;
  if (purchase?.status === "succeeded") return "Seats added.";
  if (purchase?.message) return purchase.message;
  if (["pending", "awaiting_confirmation"].includes(purchase?.status ?? "")) return "Waiting for payment confirmation.";
  return "";
}
