import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "bun:test";
import { useGooglePurchaseTracking } from "./useGooglePurchaseTracking";
import type { PurchaseInvoice } from "@/lib/googleTagManager";

const paidInvoice: PurchaseInvoice = {
  id: "hook-order",
  checkoutId: "hook-checkout",
  status: "paid",
  currency: "usd",
  netAmountCents: "2000",
  taxAmountCents: "0",
};
describe("purchase confirmation observer", () => {
  beforeEach(() => {
    Object.defineProperty(window.location, "hostname", { configurable: true, value: "app.superplane.com" });
    window.SUPERPLANE_GTM_CONTAINER_ID = "GTM-TEST123";
    delete window.dataLayer;
    localStorage.clear();
  });
  afterEach(() => {
    delete window.SUPERPLANE_GTM_CONTAINER_ID;
    Reflect.deleteProperty(window.location, "hostname");
  });
  it("waits for the matching paid invoice before pushing a conversion", async () => {
    const { rerender } = renderHook(
      ({ invoices }: { invoices: PurchaseInvoice[] }) =>
        useGooglePurchaseTracking({ checkoutID: "hook-checkout", invoices, enabled: true }),
      { initialProps: { invoices: [{ ...paidInvoice, status: "pending" }] } },
    );
    expect(window.dataLayer).toBeUndefined();
    rerender({ invoices: [paidInvoice] });
    await waitFor(() => expect(window.dataLayer?.filter((event) => event.event === "purchase")).toHaveLength(1));
    rerender({ invoices: [paidInvoice] });
    expect(window.dataLayer?.filter((event) => event.event === "purchase")).toHaveLength(1);
  });
  it("does not poll or track when disabled, including impersonation", () => {
    renderHook(() =>
      useGooglePurchaseTracking({ checkoutID: "hook-checkout", invoices: [paidInvoice], enabled: false }),
    );
    expect(window.dataLayer).toBeUndefined();
  });
  it("does not poll for a regular billing page visit", () => {
    renderHook(() => useGooglePurchaseTracking({ checkoutID: "", invoices: [paidInvoice], enabled: true }));
    expect(window.dataLayer).toBeUndefined();
  });
});
