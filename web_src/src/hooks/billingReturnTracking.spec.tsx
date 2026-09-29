import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";
import { useGooglePurchaseTracking } from "./useGooglePurchaseTracking";
import { useHostedCreditReturnRefresh } from "./useHostedCreditReturnRefresh";
import { useOrganizationBillingSync } from "./useOrganizationBillingSync";
import { rememberHostedCreditGrantSnapshot } from "@/lib/hostedCredit";
import type { PurchaseInvoice } from "@/lib/googleTagManager";

describe("shared billing return refresh", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    Object.defineProperty(window.location, "hostname", { configurable: true, value: "app.superplane.com" });
    window.SUPERPLANE_GTM_CONTAINER_ID = "GTM-TEST123";
    delete window.dataLayer;
    localStorage.clear();
    sessionStorage.clear();
  });
  afterEach(() => {
    vi.useRealTimers();
    Reflect.deleteProperty(window.location, "hostname");
    delete window.SUPERPLANE_GTM_CONTAINER_ID;
  });
  it.each(["credit", "subscription", "both"])(
    "shares one polling cycle for %s and waits for a paid order",
    async (kind) => {
      const creditAdded = kind !== "subscription";
      const subscribed = kind !== "credit";
      const sync = vi.fn().mockResolvedValue(undefined);
      const refetch = vi.fn().mockResolvedValue(undefined);
      rememberHostedCreditGrantSnapshot("org-shared", 100);
      const invoice: PurchaseInvoice = {
        id: `shared-${kind}`,
        checkoutId: `checkout-${kind}`,
        currency: "usd",
        netAmountCents: "2000",
        taxAmountCents: "0",
        status: "paid",
      };
      const { rerender, unmount } = renderHook(
        ({
          invoices,
          grantTotalCents,
          creditPurchaseAllowed,
        }: {
          invoices: PurchaseInvoice[];
          grantTotalCents: number;
          creditPurchaseAllowed: boolean;
        }) => {
          const purchasePending = useGooglePurchaseTracking({
            checkoutID: `checkout-${kind}`,
            invoices,
            enabled: true,
          });
          const creditStatus = useHostedCreditReturnRefresh({
            organizationId: "org-shared",
            creditAdded,
            grantTotalCents,
            pollingEnabled: false,
          });
          useOrganizationBillingSync({
            organizationId: "org-shared",
            subscribed,
            creditPurchaseAllowed,
            sync,
            refetch,
            returnRefreshPending: purchasePending || creditStatus === "refreshing",
          });
        },
        { initialProps: { invoices: [], grantTotalCents: 100, creditPurchaseAllowed: false } },
      );
      await act(async () => {});
      await act(async () => {
        await vi.advanceTimersByTimeAsync(2000);
      });
      expect(refetch).toHaveBeenCalledTimes(2);
      expect(window.dataLayer).toBeUndefined();
      // Credit and plan can be ready before the paid order reaches the response.
      rerender({ invoices: [], grantTotalCents: 200, creditPurchaseAllowed: true });
      await act(async () => {
        await vi.advanceTimersByTimeAsync(2000);
      });
      expect(refetch).toHaveBeenCalledTimes(3);
      rerender({ invoices: [invoice], grantTotalCents: 200, creditPurchaseAllowed: true });
      await act(async () => {
        await vi.advanceTimersByTimeAsync(4000);
      });
      expect(refetch).toHaveBeenCalledTimes(3);
      expect(window.dataLayer?.filter((event) => event.event === "purchase")).toHaveLength(1);
      rerender({ invoices: [invoice], grantTotalCents: 200, creditPurchaseAllowed: true });
      expect(window.dataLayer?.filter((event) => event.event === "purchase")).toHaveLength(1);
      unmount();
    },
  );
});
