import { useEffect } from "react";
import {
  confirmedPurchase,
  isGoogleTagManagerEnabled,
  trackGooglePurchase,
  type PurchaseInvoice,
} from "@/lib/googleTagManager";

export function useGooglePurchaseTracking({
  checkoutID,
  invoices,
  refetch,
  enabled,
}: {
  checkoutID: string;
  invoices: PurchaseInvoice[];
  refetch: () => Promise<unknown>;
  enabled: boolean;
}) {
  const confirmed = enabled && Boolean(confirmedPurchase(checkoutID, invoices));
  useEffect(() => {
    if (enabled && isGoogleTagManagerEnabled()) trackGooglePurchase(checkoutID, invoices);
  }, [enabled, checkoutID, invoices]);

  useEffect(() => {
    if (!enabled || !checkoutID || confirmed || !isGoogleTagManagerEnabled()) return;
    let stopped = false;
    let timer: ReturnType<typeof setTimeout>;
    let attempts = 0;
    const refresh = async () => {
      try {
        await refetch();
      } catch {
        /* A later poll can retry a temporary failure. */
      }
      if (!stopped && ++attempts < 30) timer = setTimeout(refresh, 2000);
    };
    void refresh();
    return () => {
      stopped = true;
      clearTimeout(timer);
    };
  }, [enabled, checkoutID, confirmed, refetch]);
}
