import { useEffect, useRef, useState } from "react";
import type { QueryClient } from "@tanstack/react-query";

import type { OrganizationsSeatPurchase } from "@/api-client";
import { isSeatPurchasePending } from "@/lib/seatPurchases";
import { syncOrganizationBilling } from "./useOrganizationBilling";

const SYNC_INTERVAL_MS = 3000;
const SYNC_TIMEOUT_MS = 120_000;

export function useSeatPurchasePolling({
  organizationId,
  enabled,
  queryClient,
  purchase,
  recordPurchase,
}: {
  organizationId: string;
  enabled: boolean;
  queryClient: QueryClient;
  purchase: OrganizationsSeatPurchase | undefined;
  recordPurchase: (purchase: OrganizationsSeatPurchase | undefined) => void;
}) {
  const [timedOut, setTimedOut] = useState(false);
  const pollingWindows = useRef(new Map<string, number>());
  const key = purchase?.id ?? purchase?.createdAt;
  const status = purchase?.status;
  const pending = isSeatPurchasePending(purchase);

  useEffect(() => {
    pollingWindows.current.clear();
    setTimedOut(false);
  }, [organizationId]);

  useEffect(() => {
    if (!pending) setTimedOut(false);
    if (!enabled || !organizationId || !pending || !key) return;
    const startedAt = pollingWindows.current.get(key) ?? Date.now();
    pollingWindows.current.set(key, startedAt);
    const remaining = SYNC_TIMEOUT_MS - (Date.now() - startedAt);
    if (remaining <= 0) {
      setTimedOut(true);
      return;
    }
    setTimedOut(false);
    let active = true;
    let inFlight = false;
    const interval = window.setInterval(async () => {
      if (!active || inFlight || Date.now() - startedAt >= SYNC_TIMEOUT_MS) return;
      inFlight = true;
      try {
        const billing = await syncOrganizationBilling(organizationId, queryClient);
        if (active) recordPurchase(billing.lastSeatPurchase);
      } catch {
        // Keep the server message. A later poll or explicit refresh can recover.
      } finally {
        inFlight = false;
      }
    }, SYNC_INTERVAL_MS);
    const timeout = window.setTimeout(() => {
      active = false;
      window.clearInterval(interval);
      setTimedOut(true);
    }, remaining);
    return () => {
      active = false;
      window.clearInterval(interval);
      window.clearTimeout(timeout);
    };
  }, [organizationId, enabled, pending, key, status, queryClient, recordPurchase]);

  return timedOut;
}
