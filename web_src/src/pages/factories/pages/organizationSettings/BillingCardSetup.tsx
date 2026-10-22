import type { useBillingCardSetup } from "@/hooks/useBillingCardSetup";

export function BillingCardSetup({ setup }: { setup: ReturnType<typeof useBillingCardSetup> }) {
  return (
    <div className="space-y-3">
      <p className="text-sm text-muted-foreground">
        Save a card securely with Polar. Card setup does not charge you or start a subscription.
      </p>
      <p role="status">
        {setup.opening
          ? "Opening card setup…"
          : setup.waitingForCard
            ? "Waiting for your saved card…"
            : "Select your plan again to reopen card setup."}
      </p>
      {setup.error && (
        <p role="alert" className="text-sm text-destructive">
          {setup.error}
        </p>
      )}
    </div>
  );
}
