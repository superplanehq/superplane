import { Button } from "@/components/ui/button";
import type { useBillingCardSetup } from "@/hooks/useBillingCardSetup";

export function BillingCardSetup({
  setup,
  disabled = false,
}: {
  setup: ReturnType<typeof useBillingCardSetup>;
  disabled?: boolean;
}) {
  return (
    <div className="space-y-3">
      <p className="text-sm text-muted-foreground">
        Save a card securely with Polar. Card setup does not charge you or start a subscription.
      </p>
      <Button type="button" disabled={disabled || setup.isPending} onClick={setup.open}>
        {setup.opening ? "Opening card setup…" : "Add payment method"}
      </Button>
      {setup.waitingForCard && <p role="status">Waiting for your saved card…</p>}
      {setup.error && (
        <p role="alert" className="text-sm text-destructive">
          {setup.error}
        </p>
      )}
    </div>
  );
}
