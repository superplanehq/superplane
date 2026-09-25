import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";
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
describe("purchase confirmation polling", () => {
  beforeEach(() => {
    Object.defineProperty(window.location, "hostname", { configurable: true, value: "app.superplane.com" });
    delete window.dataLayer;
    localStorage.clear();
  });
  afterEach(() => {
    Reflect.deleteProperty(window.location, "hostname");
  });
  it("waits for the matching paid invoice before pushing a conversion", async () => {
    const refetch = vi.fn().mockResolvedValue(undefined);
    const { rerender } = renderHook(
      ({ invoices }: { invoices: PurchaseInvoice[] }) =>
        useGooglePurchaseTracking({ checkoutID: "hook-checkout", invoices, refetch, enabled: true }),
      { initialProps: { invoices: [{ ...paidInvoice, status: "pending" }] } },
    );
    await waitFor(() => expect(refetch).toHaveBeenCalledTimes(1));
    expect(window.dataLayer).toBeUndefined();
    rerender({ invoices: [paidInvoice] });
    await waitFor(() => expect(window.dataLayer?.filter((event) => event.event === "purchase")).toHaveLength(1));
    rerender({ invoices: [paidInvoice] });
    expect(window.dataLayer?.filter((event) => event.event === "purchase")).toHaveLength(1);
  });
  it("does not poll or track when disabled, including impersonation", () => {
    const refetch = vi.fn();
    renderHook(() =>
      useGooglePurchaseTracking({ checkoutID: "hook-checkout", invoices: [paidInvoice], refetch, enabled: false }),
    );
    expect(refetch).not.toHaveBeenCalled();
    expect(window.dataLayer).toBeUndefined();
  });
  it("does not poll for a regular billing page visit", () => {
    const refetch = vi.fn();
    renderHook(() => useGooglePurchaseTracking({ checkoutID: "", invoices: [paidInvoice], refetch, enabled: true }));
    expect(refetch).not.toHaveBeenCalled();
    expect(window.dataLayer).toBeUndefined();
  });
});
