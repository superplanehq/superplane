import { BILLING_PAYMENT_BLOCKED_SEATS_COPY, billingPeriodEndLabel } from "../../lib/billingPlans";
import { Button } from "@/components/ui/button";
import { BillingSeatPurchaseLimit } from "./BillingSeatPurchaseStatus";
import { formatUsdCents } from "../../lib/workOrderUsage";
import { FactorySettingsCard } from "../settings/FactorySettingsCard";
import type { BillingPaymentBlock, BillingSeatSummary } from "./useOrganizationBillingPageModel";

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
  if (seats.paymentBlock) {
    return (
      <BlockedSeatsSummary
        seats={seats}
        paymentBlock={seats.paymentBlock}
        currentPeriodEnd={currentPeriodEnd}
        cancelAtPeriodEnd={cancelAtPeriodEnd}
        canManageBilling={canManageBilling}
      />
    );
  }
  const date = billingPeriodEndLabel(currentPeriodEnd);
  const estimate = renewalEstimate(seats, date);

  return (
    <FactorySettingsCard title="Seats" data-testid="billing-seats">
      <p className="text-xl font-semibold tracking-[-0.02em] tabular-nums">
        {seats.occupied ?? "Not available"} of {seats.purchased ?? "Not available"} seats used
      </p>
      <OverCapacityNotice
        seats={seats}
        canManageBilling={canManageBilling}
        adminAction="Buy seats or remove members to invite again."
        memberAction="Ask an organization admin to buy seats or remove members."
      />
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
            {renewalEstimateLabel({ paused: false, cancelAtPeriodEnd })}
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

type OverCapacityNoticeProps = {
  seats: BillingSeatSummary;
  canManageBilling: boolean;
  /** The recovery step for an admin, after the member and paid seat counts. */
  adminAction: string;
  /** The full message for a user who cannot manage billing. */
  memberAction: string;
};

function OverCapacityNotice({ seats, canManageBilling, adminAction, memberAction }: OverCapacityNoticeProps) {
  const { occupied, purchased } = seats;
  if (occupied === undefined || purchased === undefined || occupied <= purchased) {
    return null;
  }

  const message = canManageBilling
    ? `${pluralize(occupied, "member")}, ${pluralize(purchased, "paid seat")}. ${adminAction}`
    : memberAction;
  return (
    <>
      <p role="status" className="mt-3 text-sm text-destructive">
        {message}
      </p>
      <RenewalCapExplanation renewalSeats={seats.expectedRenewalSeats} />
    </>
  );
}

// Renewal never adds seats, so members above the paid seats are not billed.
// Without a renewal count, the estimate already asks for a repair.
function RenewalCapExplanation({ renewalSeats }: { renewalSeats?: number }) {
  if (renewalSeats === undefined) {
    return null;
  }
  return (
    <p className="mt-1 text-sm text-muted-foreground" data-testid="billing-seats-renewal-cap">
      Renewal bills {pluralize(renewalSeats, "paid seat")}. Renewal does not add seats.
    </p>
  );
}

function pluralize(count: number, noun: string) {
  return `${count} ${noun}${count === 1 ? "" : "s"}`;
}

type BlockedSeatsSummaryProps = {
  seats: BillingSeatSummary;
  paymentBlock: BillingPaymentBlock;
  currentPeriodEnd?: string;
  cancelAtPeriodEnd: boolean;
  canManageBilling: boolean;
};

// While payment is blocked, nobody can join or buy seats. The renewal still
// shows, so the admin knows what the recovered subscription charges.
function BlockedSeatsSummary({
  seats,
  paymentBlock,
  currentPeriodEnd,
  cancelAtPeriodEnd,
  canManageBilling,
}: BlockedSeatsSummaryProps) {
  const paused = paymentBlock === "paused";
  // A paused subscription does not renew until it resumes, so it has no renewal date.
  const date = paused ? null : billingPeriodEndLabel(currentPeriodEnd);
  return (
    <FactorySettingsCard title="Seats" data-testid="billing-seats">
      <p className="text-xl font-semibold tracking-[-0.02em] tabular-nums">
        {seats.occupied ?? "Not available"} of {seats.purchased ?? "Not available"} seats used
      </p>
      <p className="mt-3 text-sm text-muted-foreground" data-testid="billing-seats-blocked">
        {BILLING_PAYMENT_BLOCKED_SEATS_COPY}
      </p>
      <OverCapacityNotice
        seats={seats}
        canManageBilling={canManageBilling}
        adminAction={
          paused
            ? "Resume the subscription before you add seats or members."
            : "Update the payment method before you add seats or members."
        }
        memberAction="Ask an organization admin to update the payment."
      />
      <dl className="mt-4 grid gap-3 text-sm sm:grid-cols-2">
        {!paused && (
          <div>
            <dt className="text-xs text-muted-foreground">
              {cancelAtPeriodEnd ? "Subscription ends" : "Next renewal"}
            </dt>
            <dd className="mt-1">{date ?? "Not available"}</dd>
          </div>
        )}
        <div className="sm:col-span-2">
          <dt className="text-xs text-muted-foreground">{renewalEstimateLabel({ paused, cancelAtPeriodEnd })}</dt>
          <dd className="mt-1">{renewalEstimate(seats, date)}</dd>
        </div>
      </dl>
      <p className="mt-3 text-xs text-muted-foreground">Estimates exclude taxes and discounts.</p>
    </FactorySettingsCard>
  );
}

// Only an active subscription that is not set to end renews for sure. Other
// states show a conditional estimate.
function renewalEstimateLabel({ paused, cancelAtPeriodEnd }: { paused: boolean; cancelAtPeriodEnd: boolean }) {
  if (paused) {
    return "Renewal estimate if the subscription resumes";
  }
  return cancelAtPeriodEnd ? "Renewal estimate if you keep the subscription" : "Expected renewal";
}

// The server omits the renewal count when the paid seat count is zero or
// missing. That state needs repair, so the card shows no invented estimate.
const INVALID_PAID_SEATS_ESTIMATE = "Not available. The paid seat count needs repair. Contact support.";

function renewalEstimate(seats: BillingSeatSummary, date: string | null) {
  const renewalSeats = seats.expectedRenewalSeats;
  if (renewalSeats === undefined) {
    return INVALID_PAID_SEATS_ESTIMATE;
  }
  const renewalAmount = seatAmountLabel(seats.expectedRenewalCents);
  const renewalDate = date ? ` on ${date}` : "";
  return `${pluralize(renewalSeats, "seat")}, ${renewalAmount}${renewalDate}, based on current members.`;
}

function seatAmountLabel(cents: string | undefined) {
  if (cents === undefined || cents.trim() === "") {
    return "Not available";
  }
  const amount = Number(cents);
  return Number.isFinite(amount) ? formatUsdCents(amount) : "Not available";
}
