import { billingPeriodEndLabel } from "../../lib/billingPlans";
import { formatUsdCents } from "../../lib/workOrderUsage";
import { FactorySettingsCard } from "../settings/FactorySettingsCard";
import type { BillingSeatSummary } from "./useOrganizationBillingPageModel";

type BillingSeatsCardProps = {
  seats: BillingSeatSummary;
  currentPeriodEnd?: string;
  cancelAtPeriodEnd: boolean;
  canManageBilling: boolean;
};

export function BillingSeatsCard({
  seats,
  currentPeriodEnd,
  cancelAtPeriodEnd,
  canManageBilling,
}: BillingSeatsCardProps) {
  const date = billingPeriodEndLabel(currentPeriodEnd);
  const renewalSeats = seats.expectedRenewalSeats;
  const renewalQuantity = renewalSeats === undefined ? "Not available" : String(renewalSeats);
  const renewalAmount = seatAmountLabel(seats.expectedRenewalCents);
  const seatLabel = renewalSeats === 1 ? "seat" : "seats";
  const renewalDate = date ? ` on ${date}` : "";
  const estimate = `${renewalQuantity} ${seatLabel}, ${renewalAmount}${renewalDate}, based on current members.`;

  return (
    <FactorySettingsCard title="Seats" data-testid="billing-seats">
      <p className="text-xl font-semibold tracking-[-0.02em] tabular-nums">
        {seats.occupied ?? "Not available"} of {seats.purchased ?? "Not available"} seats used
      </p>
      <OverCapacityNotice seats={seats} canManageBilling={canManageBilling} />
      <dl className="mt-4 grid gap-3 text-sm sm:grid-cols-2">
        <div>
          <dt className="text-xs text-muted-foreground">Available seats</dt>
          <dd className="mt-1 tabular-nums">{seats.available ?? "Not available"}</dd>
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
        Unused seats are removed at renewal. Pending invitations are not renewed and expire at renewal.
      </p>
    </FactorySettingsCard>
  );
}

function OverCapacityNotice({ seats, canManageBilling }: { seats: BillingSeatSummary; canManageBilling: boolean }) {
  const { occupied, purchased } = seats;
  if (occupied === undefined || purchased === undefined || occupied <= purchased) {
    return null;
  }

  const message = canManageBilling
    ? `${pluralize(occupied, "member")}, ${pluralize(purchased, "paid seat")}. Buy seats or remove members to invite again.`
    : "Ask an organization admin to buy seats or remove members.";
  return (
    <p role="status" className="mt-3 text-sm text-destructive">
      {message}
    </p>
  );
}

function pluralize(count: number, noun: string) {
  return `${count} ${noun}${count === 1 ? "" : "s"}`;
}

function seatAmountLabel(cents: string | undefined) {
  if (cents === undefined || cents.trim() === "") {
    return "Not available";
  }
  const amount = Number(cents);
  return Number.isFinite(amount) ? formatUsdCents(amount) : "Not available";
}
