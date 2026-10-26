import { useCallback, useEffect, useRef } from "react";
import { useIsSupportSession } from "@/hooks/useSupportSession";

export const BILLING_SUBSCRIPTION_SYNC_INTERVAL_MS = 2000;
export const BILLING_SUBSCRIPTION_SYNC_TIMEOUT_MS = 30_000;

export function useOrganizationBillingSync({
  organizationId,
  subscribed,
  creditPurchaseAllowed,
  sync,
  refetch,
  returnRefreshPending = false,
}: {
  organizationId: string;
  subscribed: boolean;
  creditPurchaseAllowed: boolean;
  sync: () => Promise<unknown>;
  refetch: () => Promise<unknown>;
  returnRefreshPending?: boolean;
}) {
  const inSupportSession = useIsSupportSession();
  const inFlight = useRef<Promise<void> | null>(null);
  const run = useCallback(() => {
    if (inFlight.current) return inFlight.current;
    const refresh = async () => {
      try {
        await sync();
      } catch {
        // Keep the local billing describe result when Polar is unavailable.
      }
      try {
        await refetch();
      } catch {
        // A later cycle can retry temporary query failures.
      }
    };
    inFlight.current = refresh().finally(() => {
      inFlight.current = null;
    });
    return inFlight.current;
  }, [sync, refetch]);

  useEffect(() => {
    if (!organizationId || inSupportSession) {
      return;
    }
    void run();
  }, [organizationId, inSupportSession, run]);

  useEffect(() => {
    if ((!returnRefreshPending && (!subscribed || creditPurchaseAllowed)) || !organizationId || inSupportSession) {
      return;
    }

    const startedAt = Date.now();
    const intervalId = window.setInterval(() => {
      void run();
      if (Date.now() - startedAt >= (returnRefreshPending ? 60_000 : BILLING_SUBSCRIPTION_SYNC_TIMEOUT_MS)) {
        window.clearInterval(intervalId);
      }
    }, BILLING_SUBSCRIPTION_SYNC_INTERVAL_MS);

    return () => window.clearInterval(intervalId);
  }, [subscribed, creditPurchaseAllowed, organizationId, inSupportSession, run, returnRefreshPending]);
}
