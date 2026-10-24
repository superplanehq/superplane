import { BILLING_PAYMENT_BLOCKED_SEATS_COPY, billingPeriodEndLabel } from "../../lib/billingPlans";
import { Button } from "@/components/ui/button";
import { BillingSeatPurchaseLimit } from "./BillingSeatPurchaseStatus";
import { formatUsdCents } from "../../lib/workOrderUsage";
import { FactorySettingsCard } from "../settings/FactorySettingsCard";
import type { BillingSeatSummary } from "./useOrganizationBillingPageModel";

type BillingSeatsCardProps = {
  seats: BillingSeatSummary;
  currentPeriodEnd?: string;
  cancelAtPeriodEnd: boolean;
  canManageBilling: boolean;
  canBuySeats: boolean;
  onBuySeats: () => void;
};

export function BillingSeatsCard({
  seats,
  currentPeriodEnd,
  cancelAtPeriodEnd,
  canManageBilling,
  canBuySeats,
  onBuySeats,
}: BillingSeatsCardProps) {
  if (seats.paymentBlocked) {
    return <BlockedSeatsSummary seats={seats} />;
  }
  const date = billingPeriodEndLabel(currentPeriodEnd);
  const estimate = renewalEstimate(seats, date);

  return (
    <FactorySettingsCard title="Seats" data-testid="billing-seats">
      <p className="text-xl font-semibold tracking-[-0.02em] tabular-nums">
        {seats.occupied ?? "Not available"} of {seats.purchased ?? "Not available"} seats used
      </p>
      <dl className="mt-4 grid gap-3 text-sm sm:grid-cols-2">
        <div>
          <dt className="text-xs text-muted-foreground">Available seats</dt>
          <dd className="mt-1 tabular-nums">{seats.available ?? "Not available"}</dd>
        </div>
        <div>
          <dt className="text-xs text-muted-foreground">Pending invitations</dt>
          <dd className="mt-1 tabular-nums">{seats.pendingInvitations ?? "Not available"}</dd>
        </div>
        <div>
          <dt className="text-xs text-muted-foreground">Seats available for new invitations</dt>
          <dd className="mt-1 tabular-nums">{seats.invitable ?? "Unlimited"}</dd>
        </div>
        <div>
          <dt className="text-xs text-muted-foreground">Plan maximum</dt>
          <dd className="mt-1 tabular-nums">{seats.planMaximum ?? "Unlimited"}</dd>
        </div>
        <div>
          <dt className="text-xs text-muted-foreground">Current monthly total</dt>
          <dd className="mt-1 tabular-nums">{seatAmountLabel(seats.currentMonthlyTotalCents)}</dd>
        </div>
        <div>
          <dt className="text-xs text-muted-foreground">{cancelAtPeriodEnd ? "Subscription ends" : "Next renewal"}</dt>
          <dd className="mt-1">{date ?? "Not available"}</dd>
        </div>
        <div className="sm:col-span-2">
          <dt className="text-xs text-muted-foreground">
            {cancelAtPeriodEnd ? "Renewal estimate if you keep the subscription" : "Expected renewal"}
          </dt>
          <dd className="mt-1">{estimate}</dd>
        </div>
      </dl>
      <p className="mt-3 text-xs text-muted-foreground">Estimates exclude taxes and discounts.</p>
      <p className="mt-2 text-xs text-muted-foreground">
        Pending invitations do not count at renewal. After renewal, an invitation needs a free seat.
      </p>
      <div className="mt-4 space-y-3">
        {!canManageBilling && <p className="text-sm text-muted-foreground">Ask an organization admin to buy seats.</p>}
        {canBuySeats &&
          !cancelAtPeriodEnd &&
          (seats.purchasable === 0 ? (
            <BillingSeatPurchaseLimit maximum={seats.planMaximum} />
          ) : (
            <Button onClick={onBuySeats}>Buy seats</Button>
          ))}
      </div>
    </FactorySettingsCard>
  );
}

// While payment is blocked, the server sends only the stored and used seats.
function BlockedSeatsSummary({ seats }: { seats: BillingSeatSummary }) {
  return (
    <FactorySettingsCard title="Seats" data-testid="billing-seats">
      <p className="text-xl font-semibold tracking-[-0.02em] tabular-nums">
        {seats.occupied ?? "Not available"} of {seats.purchased ?? "Not available"} seats used
      </p>
      <p className="mt-3 text-sm text-muted-foreground" data-testid="billing-seats-blocked">
        {BILLING_PAYMENT_BLOCKED_SEATS_COPY}
      </p>
    </FactorySettingsCard>
  );
}

function renewalEstimate(seats: BillingSeatSummary, date: string | null) {
  const renewalSeats = seats.expectedRenewalSeats;
  const renewalQuantity = renewalSeats === undefined ? "Not available" : String(renewalSeats);
  const renewalAmount = seatAmountLabel(seats.expectedRenewalCents);
  const seatLabel = renewalSeats === 1 ? "seat" : "seats";
  const renewalDate = date ? ` on ${date}` : "";
  return `${renewalQuantity} ${seatLabel}, ${renewalAmount}${renewalDate}, based on current members.`;
}

function seatAmountLabel(cents: string | undefined) {
  if (cents === undefined || cents.trim() === "") {
    return "Not available";
  }
  const amount = Number(cents);
  return Number.isFinite(amount) ? formatUsdCents(amount) : "Not available";
}
