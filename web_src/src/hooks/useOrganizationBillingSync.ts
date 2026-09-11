import { useCallback, useEffect } from "react";

export const BILLING_SUBSCRIPTION_SYNC_INTERVAL_MS = 2000;
export const BILLING_SUBSCRIPTION_SYNC_TIMEOUT_MS = 30_000;

export function useOrganizationBillingSync({
  organizationId,
  subscribed,
  creditPurchaseAllowed,
  sync,
  refetch,
}: {
  organizationId: string;
  subscribed: boolean;
  creditPurchaseAllowed: boolean;
  sync: () => Promise<unknown>;
  refetch: () => Promise<unknown>;
}) {
  const run = useCallback(async () => {
    try {
      await sync();
    } catch {
      // Keep the local billing describe result when Polar is unavailable.
    }
    await refetch();
  }, [sync, refetch]);

  useEffect(() => {
    if (!organizationId) {
      return;
    }
    void run();
  }, [organizationId, run]);

  useEffect(() => {
    if (!subscribed || creditPurchaseAllowed || !organizationId) {
      return;
    }

    const startedAt = Date.now();
    const intervalId = window.setInterval(() => {
      void run();
      if (Date.now() - startedAt >= BILLING_SUBSCRIPTION_SYNC_TIMEOUT_MS) {
        window.clearInterval(intervalId);
      }
    }, BILLING_SUBSCRIPTION_SYNC_INTERVAL_MS);

    return () => window.clearInterval(intervalId);
  }, [subscribed, creditPurchaseAllowed, organizationId, run]);
}
