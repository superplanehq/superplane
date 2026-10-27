import { useId, useState } from "react";
import type { OrganizationsBillingPlanOption } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { checkoutSeatBounds, type CheckoutSeatBounds } from "../../lib/billingPlans";
import { formatUsdCents } from "../../lib/workOrderUsage";

export function BillingSeatCheckout({
  plan,
  occupied,
  pending,
  disabled,
  onSubscribe,
}: {
  plan: OrganizationsBillingPlanOption;
  occupied: number | undefined;
  pending: number | undefined;
  disabled: boolean;
  onSubscribe: (seats: number) => void;
}) {
  const inputId = useId();
  const maximum = plan.seatMaximum;
  const bounds = checkoutSeatBounds(occupied, pending, maximum);
  const minimum = bounds.minimum;
  const upperBound = maximum ?? Infinity;
  // Seed once at mount. A background refresh must not overwrite the selection.
  const [value, setValue] = useState(() => String(bounds.defaultSeats));
  const seats = Number(value);
  const valid = occupied != null && value !== "" && Number.isInteger(seats) && seats >= minimum && seats <= upperBound;
  const total = valid ? formatUsdCents(seats * Number(plan.priceCents ?? 0)) : "—";
  return (
    <div className="space-y-3">
      <Label htmlFor={inputId}>{plan.name} seats</Label>
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          aria-label={`Decrease ${plan.name} seats`}
          disabled={disabled || !Number.isInteger(seats) || seats <= minimum}
          onClick={() => setValue(String(seats - 1))}
        >
          −
        </Button>
        <Input
          id={inputId}
          type="number"
          min={minimum}
          max={maximum}
          step={1}
          value={value}
          disabled={disabled}
          aria-invalid={!valid}
          onChange={(event) => setValue(event.target.value)}
          className="w-24"
        />
        <Button
          variant="outline"
          aria-label={`Increase ${plan.name} seats`}
          disabled={disabled || !Number.isInteger(seats) || seats >= upperBound}
          onClick={() => setValue(String(seats + 1))}
        >
          +
        </Button>
      </div>
      <BillingSeatBounds occupied={occupied} bounds={bounds} maximum={maximum} />
      <div className="flex items-center justify-between gap-3 rounded-md bg-muted/40 p-4">
        <p className="text-sm">Monthly total</p>
        <p className="text-xl font-semibold tabular-nums">{total}</p>
      </div>
      <p className="text-xs text-muted-foreground">Estimates exclude tax.</p>
      <Button className="w-full" disabled={disabled || !valid} onClick={() => onSubscribe(seats)}>
        Subscribe to {plan.name}
      </Button>
    </div>
  );
}

function BillingSeatBounds({
  occupied,
  bounds,
  maximum,
}: {
  occupied: number | undefined;
  bounds: CheckoutSeatBounds;
  maximum: number | undefined;
}) {
  const { minimum, coveredInvitations, invitationsWithoutSeat } = bounds;
  if (occupied == null) {
    return <p role="alert">Could not load seat usage. Refresh billing settings and try again.</p>;
  }
  if (maximum != null && maximum < minimum) {
    return <p role="alert">Current members exceed this plan's seat maximum. Contact an organization admin.</p>;
  }
  return (
    <div className="space-y-1 text-sm">
      <p className="text-muted-foreground">
        {maximum == null ? `Select at least ${minimum} seats.` : `Select ${minimum} to ${maximum} seats.`}
      </p>
      {coveredInvitations > 0 && (
        <p className="text-muted-foreground">
          The default includes {countOf(coveredInvitations, "seat")} for pending invitations. You can remove these
          seats.
        </p>
      )}
      {maximum != null && invitationsWithoutSeat > 0 && (
        <p role="status">
          The plan maximum is {countOf(maximum, "seat")}. {countOf(invitationsWithoutSeat, "pending invitation")} cannot
          get a seat.
        </p>
      )}
    </div>
  );
}

function countOf(count: number, noun: string) {
  return `${count} ${noun}${count === 1 ? "" : "s"}`;
}
