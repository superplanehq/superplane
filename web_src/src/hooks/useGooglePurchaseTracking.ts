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
  enabled,
}: {
  checkoutID: string;
  invoices: PurchaseInvoice[];
  enabled: boolean;
}) {
  const confirmed = enabled && Boolean(confirmedPurchase(checkoutID, invoices));
  useEffect(() => {
    if (enabled && isGoogleTagManagerEnabled()) trackGooglePurchase(checkoutID, invoices);
  }, [enabled, checkoutID, invoices]);

  return enabled && isGoogleTagManagerEnabled() && Boolean(checkoutID) && !confirmed;
}
