import { useId, useState } from "react";
import type { OrganizationsBillingPlanOption } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { formatUsdCents } from "../../lib/workOrderUsage";

export function BillingSeatCheckout({
  plan,
  occupied,
  disabled,
  onSubscribe,
}: {
  plan: OrganizationsBillingPlanOption;
  occupied: number | undefined;
  disabled: boolean;
  onSubscribe: (seats: number) => void;
}) {
  const inputId = useId();
  const minimum = Math.max(occupied ?? 0, 1);
  const maximum = plan.seatMaximum;
  const upperBound = maximum ?? Infinity;
  const [value, setValue] = useState(String(minimum));
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
      <BillingSeatBounds occupied={occupied} minimum={minimum} maximum={maximum} />
      <div className="text-sm">
        <p>Initial payment estimate: {total}</p>
        <p>Monthly total: {total}</p>
        <p className="text-muted-foreground">Estimates exclude tax.</p>
      </div>
      <Button variant="outline" disabled={disabled || !valid} onClick={() => onSubscribe(seats)}>
        Subscribe to {plan.name}
      </Button>
    </div>
  );
}

function BillingSeatBounds({
  occupied,
  minimum,
  maximum,
}: {
  occupied: number | undefined;
  minimum: number;
  maximum: number | undefined;
}) {
  if (occupied == null) {
    return <p role="alert">Could not load seat usage. Refresh billing settings and try again.</p>;
  }
  if (maximum != null && maximum < minimum) {
    return <p role="alert">Current members exceed this plan's seat maximum. Contact an organization admin.</p>;
  }
  return (
    <p className="text-sm text-muted-foreground">
      {maximum == null ? `Select at least ${minimum} seats.` : `Select ${minimum} to ${maximum} seats.`}
    </p>
  );
}
