import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import type { OrganizationsSeatPurchase } from "@/api-client";
import { client } from "@/api-client/client.gen";
import { useOrganizationBilling } from "./useOrganizationBilling";
import { useSeatPurchase } from "./useSeatPurchase";

type PurchaseBody = { additionalSeats?: number; expectedPurchasedSeats?: number; operationKey?: string };
// A number is an HTTP status that the server returns before it processes the request.
type Fault = "lost request" | "lost response" | number;

const DAY_MS = 24 * 60 * 60 * 1000;

// FakeBillingServer follows the purchase contract of the server: the
// operation key with the same request replays the stored purchase, also after
// a renewal. The same key with another request gets HTTP 412, a missing key
// HTTP 400, and a stale baseline HTTP 409.
class FakeBillingServer {
  purchasedSeats = 4;
  periodStart = new Date(Date.now() - 10 * DAY_MS).toISOString();
  periodEnd = new Date(Date.now() + 20 * DAY_MS).toISOString();
  lastSeatPurchase: OrganizationsSeatPurchase | undefined;
  charges = 0;
  billingUnavailable = false;
  purchaseRequests: PurchaseBody[] = [];
  private nextFault: Fault | undefined;
  private operations = new Map<string, { fingerprint: string; purchase: OrganizationsSeatPurchase }>();

  failNextPurchase(fault: Fault) {
    this.nextFault = fault;
  }

  renew() {
    this.periodStart = this.periodEnd;
    this.periodEnd = new Date(Date.parse(this.periodEnd) + 30 * DAY_MS).toISOString();
    this.purchasedSeats = 4;
  }

  billing() {
    return {
      purchasedSeats: this.purchasedSeats,
      purchasableSeats: 10,
      currentPeriodStart: this.periodStart,
      currentPeriodEnd: this.periodEnd,
      lastSeatPurchase: this.lastSeatPurchase,
    };
  }

  fetch: typeof fetch = async (input) => {
    const request = input instanceof Request ? input : new Request(input);
    const path = new URL(request.url).pathname;
    if (path.endsWith("/seats/preview")) {
      const { additionalSeats = 0 } = (await request.json()) as PurchaseBody;
      return this.respond(200, {
        purchasedSeats: this.purchasedSeats,
        targetSeats: this.purchasedSeats + additionalSeats,
        proratedChargeCents: "500",
        periodStart: new Date().toISOString(),
        periodEnd: this.periodEnd,
      });
    }
    if (path.endsWith("/seats/purchase")) return this.purchase((await request.json()) as PurchaseBody);
    if (path.endsWith("/billing/sync") || path.endsWith("/billing")) {
      if (this.billingUnavailable) throw new TypeError("Failed to fetch");
      return this.respond(200, this.billing());
    }
    throw new Error(`Unexpected billing request: ${path}`);
  };

  private purchase(body: PurchaseBody) {
    this.purchaseRequests.push(body);
    const fault = this.nextFault;
    this.nextFault = undefined;
    if (fault === "lost request") throw new TypeError("Failed to fetch");
    if (typeof fault === "number") return this.respond(fault, { message: "The purchase was not processed." });
    const response = this.process(body);
    if (fault === "lost response") throw new TypeError("Failed to fetch");
    return response;
  }

  private process({ additionalSeats, expectedPurchasedSeats, operationKey }: PurchaseBody) {
    if (!operationKey || !additionalSeats) return this.respond(400, { message: "The request is not valid." });
    const fingerprint = `${additionalSeats}:${expectedPurchasedSeats}`;
    const stored = this.operations.get(operationKey);
    if (stored && stored.fingerprint !== fingerprint)
      return this.respond(412, { message: "The operation key belongs to another purchase." });
    if (stored) return this.respond(200, { purchase: stored.purchase, billing: this.billing() });
    if (expectedPurchasedSeats !== this.purchasedSeats)
      return this.respond(409, { message: "The seat count changed. Preview this purchase again." });
    this.charges += 1;
    const purchase = {
      id: `purchase-${this.charges}`,
      status: "succeeded",
      fromSeats: this.purchasedSeats,
      toSeats: this.purchasedSeats + additionalSeats,
      createdAt: new Date().toISOString(),
    };
    this.purchasedSeats = purchase.toSeats;
    this.lastSeatPurchase = purchase;
    this.operations.set(operationKey, { fingerprint, purchase });
    return this.respond(200, { purchase, billing: this.billing() });
  }

  private respond(status: number, body: unknown) {
    return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
  }
}

// Each mount has its own query cache, like a page load.
function mountPage(organizationId = "org-1") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  }
  return renderHook(
    () => {
      const billing = useOrganizationBilling(organizationId);
      return useSeatPurchase({ organizationId, billing: billing.data, enabled: true });
    },
    { wrapper: Wrapper },
  );
}

async function advance(milliseconds: number) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(milliseconds);
  });
}

async function loadPage(organizationId = "org-1") {
  const page = mountPage(organizationId);
  await advance(0);
  return page;
}

async function previewAndConfirm(page: ReturnType<typeof mountPage>) {
  if (page.result.current.isOpen) act(() => page.result.current.requestPreview());
  else act(() => page.result.current.setOpen(true));
  await advance(300);
  expect(page.result.current.canConfirm).toBe(true);
  await act(async () => page.result.current.confirm());
}

describe("seat purchase HTTP recovery", () => {
  const previousConfig = client.getConfig();
  let server: FakeBillingServer;

  beforeEach(() => {
    vi.useFakeTimers();
    window.localStorage.clear();
    server = new FakeBillingServer();
    client.setConfig({ baseUrl: "http://localhost", fetch: server.fetch });
  });

  afterEach(() => {
    cleanup();
    client.setConfig({ ...previousConfig, fetch: previousConfig.fetch });
    vi.useRealTimers();
  });

  it("uses an HTTP 409 response to refresh the baseline and require a new confirmation", async () => {
    const page = await loadPage();
    act(() => page.result.current.setOpen(true));
    await advance(300);
    expect(page.result.current.preview?.purchasedSeats).toBe(4);
    server.purchasedSeats = 7;
    await act(async () => page.result.current.confirm());
    expect(page.result.current.preview).toBeUndefined();
    expect(server.purchaseRequests).toEqual([
      { additionalSeats: 1, expectedPurchasedSeats: 4, operationKey: expect.stringMatching(/^[0-9a-f-]{36}$/) },
    ]);
    await advance(300);
    expect(page.result.current.preview?.purchasedSeats).toBe(7);
    expect(page.result.current.canConfirm).toBe(true);
    expect(server.purchaseRequests).toHaveLength(1);
  });

  it.each([
    { fault: "lost request" as const, reused: true },
    { fault: 503, reused: true },
    { fault: 400, reused: false },
    { fault: 412, reused: false },
  ])("retries after $fault with the same key only while the outcome is unknown", async ({ fault, reused }) => {
    const page = await loadPage();
    server.failNextPurchase(fault);
    await previewAndConfirm(page);
    expect(page.result.current.submissionUnresolved).toBe(reused);
    await previewAndConfirm(page);
    const [first, retry] = server.purchaseRequests;
    expect({ ...retry, operationKey: undefined }).toEqual({ ...first, operationKey: undefined });
    expect(retry.operationKey === first.operationKey).toBe(reused);
    expect(server.charges).toBe(1);
    expect(page.result.current.purchaseMessage).toBe("Seats added.");
  });

  it("restores the unresolved purchase after a refresh without submitting until the user confirms", async () => {
    const first = await loadPage();
    server.failNextPurchase("lost request");
    await previewAndConfirm(first);
    first.unmount();

    const restored = await loadPage();
    expect(restored.result.current.submissionUnresolved).toBe(true);
    await advance(10_000);
    expect(server.purchaseRequests).toHaveLength(1);
    await previewAndConfirm(restored);
    expect(server.purchaseRequests[1]).toEqual(server.purchaseRequests[0]);
    expect(server.charges).toBe(1);
    expect(restored.result.current.purchaseMessage).toBe("Seats added.");
  });

  it("replays the stored purchase after a renewal restores the original seat count", async () => {
    const first = await loadPage();
    server.failNextPurchase("lost response");
    await previewAndConfirm(first);
    expect(server.charges).toBe(1);
    first.unmount();
    server.renew();

    const restored = await loadPage();
    await previewAndConfirm(restored);
    expect(server.purchaseRequests[1]).toEqual(server.purchaseRequests[0]);
    expect(server.charges).toBe(1);
    expect(restored.result.current.purchase?.id).toBe("purchase-1");
    expect(restored.result.current.attemptSucceeded).toBe(true);
    expect(restored.result.current.purchaseMessage).toBe("Seats added.");
  });

  it("charges a new purchase of the same request after a confirmed success and a renewal", async () => {
    const first = await loadPage();
    await previewAndConfirm(first);
    first.unmount();
    server.renew();

    const next = await loadPage();
    await previewAndConfirm(next);
    expect(server.purchaseRequests[1].operationKey).not.toBe(server.purchaseRequests[0].operationKey);
    expect(server.charges).toBe(2);
    expect(next.result.current.purchase?.id).toBe("purchase-2");
  });

  it.each([
    { change: "quantity", additionalSeats: "2", baseline: 4 },
    { change: "baseline", additionalSeats: "1", baseline: 6 },
  ])(
    "retains each unresolved key after a changed $change request and retires only the replayed key",
    async ({ additionalSeats, baseline }) => {
      const firstPage = await loadPage();
      server.billingUnavailable = true;
      server.failNextPurchase("lost response");
      await previewAndConfirm(firstPage);
      expect(server.charges).toBe(1);
      firstPage.unmount();
      server.renew();
      server.billingUnavailable = false;
      server.purchasedSeats = baseline;

      const changedPage = await loadPage();
      act(() => changedPage.result.current.setOpen(true));
      act(() => changedPage.result.current.setAdditionalSeats(additionalSeats));
      server.billingUnavailable = true;
      server.failNextPurchase("lost request");
      await previewAndConfirm(changedPage);
      const [first, changed] = server.purchaseRequests;
      expect(changed.operationKey).not.toBe(first.operationKey);
      changedPage.unmount();
      server.billingUnavailable = false;
      server.purchasedSeats = 4;

      const retryPage = await loadPage();
      await previewAndConfirm(retryPage);
      expect(server.purchaseRequests[2]).toEqual(first);
      expect(server.charges).toBe(1);
      expect(retryPage.result.current.purchase?.id).toBe("purchase-1");
      retryPage.unmount();
      server.purchasedSeats = baseline;

      const remainingPage = await loadPage();
      act(() => remainingPage.result.current.setOpen(true));
      act(() => remainingPage.result.current.setAdditionalSeats(additionalSeats));
      await previewAndConfirm(remainingPage);
      expect(server.purchaseRequests[3]).toEqual(changed);
      expect(server.charges).toBe(2);
    },
  );

  it("never sends the unresolved key of one organization for another organization", async () => {
    const first = await loadPage();
    server.failNextPurchase("lost request");
    await previewAndConfirm(first);
    first.unmount();

    const other = await loadPage("org-2");
    expect(other.result.current.submissionUnresolved).toBe(false);
    server.failNextPurchase("lost request");
    await previewAndConfirm(other);
    other.unmount();

    const restored = await loadPage();
    await previewAndConfirm(restored);
    const [original, otherOrganization, retry] = server.purchaseRequests.map((body) => body.operationKey);
    expect(otherOrganization).not.toBe(original);
    expect(retry).toBe(original);
  });

  it("preserves an unresolved key from the earlier single-operation storage format when another request is confirmed", async () => {
    const operationKey = crypto.randomUUID();
    window.localStorage.setItem(
      "superplane:seat-purchase-operation:org-1",
      JSON.stringify({
        version: 1,
        organizationId: "org-1",
        operationKey,
        additionalSeats: 1,
        expectedPurchasedSeats: 4,
        confirmedAt: new Date().toISOString(),
      }),
    );
    const page = await loadPage();
    act(() => page.result.current.setOpen(true));
    act(() => page.result.current.setAdditionalSeats("2"));
    server.failNextPurchase("lost request");
    await previewAndConfirm(page);
    page.unmount();

    const restored = await loadPage();
    await previewAndConfirm(restored);
    expect(server.purchaseRequests[1]).toEqual({ additionalSeats: 1, expectedPurchasedSeats: 4, operationKey });
    expect(server.charges).toBe(1);
  });

  it.each(["all storage", "writes only"])("reuses the key when the browser blocks %s", async (blocked) => {
    const storage = Object.getOwnPropertyDescriptor(window, "localStorage");
    const localStorage = window.localStorage;
    Object.defineProperty(window, "localStorage", {
      configurable: true,
      get: () => {
        if (blocked === "writes only")
          return {
            getItem: localStorage.getItem.bind(localStorage),
            removeItem: localStorage.removeItem.bind(localStorage),
            setItem: () => {
              throw new DOMException("Storage is full.", "QuotaExceededError");
            },
          };
        throw new DOMException("Storage is blocked.", "SecurityError");
      },
    });
    try {
      const page = await loadPage();
      server.billingUnavailable = true;
      server.failNextPurchase("lost response");
      await previewAndConfirm(page);
      server.renew();
      server.billingUnavailable = false;
      await act(async () => page.result.current.refreshBilling());
      await previewAndConfirm(page);
      expect(server.purchaseRequests[1]).toEqual(server.purchaseRequests[0]);
      expect(server.charges).toBe(1);
      // A terminal result must retire the fallback key as well.
      server.renew();
      await act(async () => page.result.current.refreshBilling());
      await previewAndConfirm(page);
      expect(server.purchaseRequests[2].operationKey).not.toBe(server.purchaseRequests[0].operationKey);
      expect(server.charges).toBe(2);
    } finally {
      if (storage) Object.defineProperty(window, "localStorage", storage);
    }
  });

  it("requires a new preview after the billing period renews", async () => {
    const page = await loadPage();
    act(() => page.result.current.setOpen(true));
    await advance(300);
    expect(page.result.current.canConfirm).toBe(true);
    server.renew();
    await act(async () => page.result.current.refreshBilling());
    expect(page.result.current.canConfirm).toBe(false);
    await act(async () => page.result.current.confirm());
    expect(server.purchaseRequests).toHaveLength(0);
    await advance(300);
    await act(async () => page.result.current.confirm());
    expect(server.purchaseRequests).toHaveLength(1);
  });
});
