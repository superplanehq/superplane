import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { client } from "@/api-client/client.gen";
import { useOrganizationBilling } from "./useOrganizationBilling";
import { useSeatPurchase } from "./useSeatPurchase";
import { DAY_MS, FakeBillingServer } from "./useSeatPurchase.transport.spec.fixtures";

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

// This tab cannot write storage until unblockWrites. The returned storage is
// the shared same-origin storage that other tabs still write.
function blockStorageWrites() {
  const descriptor = Object.getOwnPropertyDescriptor(window, "localStorage");
  const localStorage = window.localStorage;
  let writesBlocked = true;
  Object.defineProperty(window, "localStorage", {
    configurable: true,
    value: {
      getItem: localStorage.getItem.bind(localStorage),
      removeItem: localStorage.removeItem.bind(localStorage),
      setItem: (key: string, value: string) => {
        if (writesBlocked) throw new DOMException("Storage is full.", "QuotaExceededError");
        localStorage.setItem(key, value);
      },
    },
  });
  return {
    localStorage,
    unblockWrites: () => {
      writesBlocked = false;
    },
    restore: () => {
      if (descriptor) Object.defineProperty(window, "localStorage", descriptor);
    },
  };
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
      {
        additionalSeats: 1,
        expectedPurchasedSeats: 4,
        operationKey: expect.stringMatching(/^[0-9a-f-]{36}$/),
        ...server.confirmedContext(),
      },
    ]);
    await advance(300);
    expect(page.result.current.preview?.purchasedSeats).toBe(7);
    expect(page.result.current.canConfirm).toBe(true);
    expect(server.purchaseRequests).toHaveLength(1);
  });

  it("sends the exact subscription and billing period of the confirmed preview", async () => {
    // Microseconds show that the browser sends the server value unchanged.
    server.periodStart = new Date(Date.now() - 10 * DAY_MS).toISOString().replace("Z", "456Z");
    const page = await loadPage();
    await previewAndConfirm(page);
    expect(server.purchaseRequests).toEqual([
      {
        additionalSeats: 1,
        expectedPurchasedSeats: 4,
        operationKey: expect.any(String),
        ...server.confirmedContext(),
      },
    ]);
    expect(server.charges).toBe(1);
  });

  it("does not confirm a preview without its subscription and billing period", async () => {
    server.previewOmitsBinding = true;
    const page = await loadPage();
    act(() => page.result.current.setOpen(true));
    await advance(300);
    expect(page.result.current.canConfirm).toBe(false);
    expect(page.result.current.previewError).toBe("SuperPlane could not estimate this purchase. Try again.");
    await act(async () => page.result.current.confirm());
    expect(server.purchaseRequests).toHaveLength(0);
  });

  it.each([
    { change: "renewal with the same seat count", apply: (fake: FakeBillingServer) => fake.renew() },
    {
      change: "replacement subscription",
      apply: (fake: FakeBillingServer) => {
        fake.subscriptionId = "sub_replacement";
      },
    },
  ])("uses HTTP 409 after a $change to require a new preview and confirmation", async ({ apply }) => {
    const page = await loadPage();
    act(() => page.result.current.setOpen(true));
    await advance(300);
    const stale = server.confirmedContext();
    apply(server);
    await act(async () => page.result.current.confirm());
    expect(server.purchaseRequests).toEqual([expect.objectContaining(stale)]);
    expect(server.charges).toBe(0);
    expect(page.result.current.preview).toBeUndefined();
    await advance(10_000);
    expect(page.result.current.canConfirm).toBe(true);
    expect(server.purchaseRequests).toHaveLength(1);

    await act(async () => page.result.current.confirm());
    const [first, second] = server.purchaseRequests;
    expect(second.operationKey).not.toBe(first.operationKey);
    expect(second).toEqual(expect.objectContaining(server.confirmedContext()));
    expect(server.charges).toBe(1);
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
    const original = server.confirmedContext();
    server.failNextPurchase("lost response");
    await previewAndConfirm(first);
    expect(server.charges).toBe(1);
    first.unmount();
    server.renew();

    const restored = await loadPage();
    await previewAndConfirm(restored);
    expect(restored.result.current.preview).toBeUndefined();
    expect(server.purchaseRequests[1]).toEqual(server.purchaseRequests[0]);
    expect(server.purchaseRequests[1]).toEqual(expect.objectContaining(original));
    expect(server.purchaseRequests[1].confirmedPeriodStart).not.toBe(server.periodStart);
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

  it("replays a legacy version 1 operation without a binding when another request is confirmed", async () => {
    const operationKey = crypto.randomUUID();
    window.localStorage.setItem(
      "superplane:seat-purchase-operation:org-1",
      JSON.stringify({
        version: 1,
        organizationId: "org-1",
        operationKey,
        additionalSeats: 1,
        expectedPurchasedSeats: 4,
        // A version 1 period start is the coverage start.
        periodStart: new Date().toISOString(),
        periodEnd: server.periodEnd,
        confirmedAt: new Date().toISOString(),
      }),
    );
    // The legacy request reached Polar. A renewal restored the seat count.
    server.recordLegacyPurchase(operationKey, 1, 4);
    server.renew();
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
    expect(restored.result.current.purchase?.id).toBe("purchase-1");
  });

  it("does not charge for a legacy version 1 operation that the server never recorded", async () => {
    const operationKey = crypto.randomUUID();
    window.localStorage.setItem(
      "superplane:seat-purchase-operation:org-1",
      JSON.stringify([
        {
          version: 1,
          organizationId: "org-1",
          operationKey,
          additionalSeats: 1,
          expectedPurchasedSeats: 4,
          confirmedAt: new Date().toISOString(),
        },
      ]),
    );
    const page = await loadPage();
    await previewAndConfirm(page);
    expect(server.purchaseRequests).toEqual([{ additionalSeats: 1, expectedPurchasedSeats: 4, operationKey }]);
    expect(server.charges).toBe(0);

    await previewAndConfirm(page);
    expect(server.purchaseRequests[1].operationKey).not.toBe(operationKey);
    expect(server.purchaseRequests[1]).toEqual(expect.objectContaining(server.confirmedContext()));
    expect(server.charges).toBe(1);
  });

  it("merges another tab's unresolved key with failed writes and keeps retired keys out of later saves", async () => {
    const organizationId = "org-cross-tab-storage";
    const storageKey = `superplane:seat-purchase-operation:${organizationId}`;
    const { localStorage, unblockWrites, restore } = blockStorageWrites();
    try {
      const page = await loadPage(organizationId);
      server.failNextPurchase("lost request");
      await previewAndConfirm(page);
      const first = server.purchaseRequests[0];

      // Another tab has independent memory and writes this shared storage record
      // before sending its purchase through the same HTTP contract.
      const otherRequest = { additionalSeats: 2, expectedPurchasedSeats: 4, operationKey: crypto.randomUUID() };
      const otherBinding = {
        subscriptionId: server.subscriptionId,
        periodStart: server.periodStart,
        periodEnd: server.periodEnd,
        removal: { seatChangeVersion: "0", pendingRemovalId: "", pendingRemovalRevision: 0 },
      };
      const other = { ...otherRequest, ...server.confirmedContext() };
      localStorage.setItem(
        storageKey,
        JSON.stringify([
          {
            version: 3,
            organizationId,
            confirmedAt: new Date().toISOString(),
            ...otherRequest,
            binding: otherBinding,
          },
        ]),
      );
      server.failNextPurchase("lost response");
      await expect(
        server.fetch(
          new Request(`http://localhost/api/v1/organizations/${organizationId}/billing/seats/purchase`, {
            method: "POST",
            body: JSON.stringify(other),
          }),
        ),
      ).rejects.toThrow("Failed to fetch");
      expect(server.charges).toBe(1);
      server.renew();
      await act(async () => page.result.current.refreshBilling());

      act(() => page.result.current.setAdditionalSeats("2"));
      await previewAndConfirm(page);
      expect(server.purchaseRequests[2]).toEqual(other);
      expect(server.charges).toBe(1);

      // Retirement also fails to persist. A later successful save must preserve
      // the first tab's key without restoring the retired key from storage.
      unblockWrites();
      act(() => page.result.current.setAdditionalSeats("3"));
      server.failNextPurchase("lost request");
      await previewAndConfirm(page);
      act(() => page.result.current.setAdditionalSeats("1"));
      await previewAndConfirm(page);
      expect(server.purchaseRequests[4]).toEqual(first);
      // The first request never reached the server, and its period ended.
      expect(server.charges).toBe(1);
      await previewAndConfirm(page);
      expect(server.purchaseRequests[5].operationKey).not.toBe(first.operationKey);
      expect(server.charges).toBe(2);

      server.renew();
      await act(async () => page.result.current.refreshBilling());
      act(() => page.result.current.setAdditionalSeats("2"));
      await previewAndConfirm(page);
      expect(server.purchaseRequests[6].operationKey).not.toBe(other.operationKey);
      expect(server.charges).toBe(3);
    } finally {
      restore();
    }
  });

  it("charges a new confirmation after another tab settles the key that a failed write in this tab saw", async () => {
    const organizationId = "org-cross-tab-settled";
    const storageKey = `superplane:seat-purchase-operation:${organizationId}`;
    const { localStorage, unblockWrites, restore } = blockStorageWrites();
    try {
      // Another tab stores its confirmed purchase before it sends the request.
      const other = { additionalSeats: 2, expectedPurchasedSeats: 4, operationKey: crypto.randomUUID() };
      localStorage.setItem(
        storageKey,
        JSON.stringify([
          {
            version: 3,
            organizationId,
            confirmedAt: new Date().toISOString(),
            ...other,
            binding: {
              subscriptionId: server.subscriptionId,
              periodStart: server.periodStart,
              periodEnd: server.periodEnd,
              removal: { seatChangeVersion: "0", pendingRemovalId: "", pendingRemovalRevision: 0 },
            },
          },
        ]),
      );
      const otherBody = JSON.stringify({ ...other, ...server.confirmedContext() });

      // This tab cannot save its own confirmation, and its request is lost.
      const page = await loadPage(organizationId);
      server.failNextPurchase("lost request");
      await previewAndConfirm(page);
      expect(server.charges).toBe(0);

      // The other tab gets its answer and forgets its settled key.
      const otherResponse = await server.fetch(
        new Request(`http://localhost/api/v1/organizations/${organizationId}/billing/seats/purchase`, {
          method: "POST",
          body: otherBody,
        }),
      );
      expect(otherResponse.status).toBe(200);
      expect(server.charges).toBe(1);
      localStorage.removeItem(storageKey);

      // A new period starts at the same seat count. The user confirms the
      // same counts again as a new purchase.
      unblockWrites();
      server.renew();
      await act(async () => page.result.current.refreshBilling());
      act(() => page.result.current.setAdditionalSeats("2"));
      await previewAndConfirm(page);
      const retry = server.purchaseRequests[2];
      expect(retry.operationKey).not.toBe(other.operationKey);
      expect(retry).toEqual(expect.objectContaining({ ...server.confirmedContext(), additionalSeats: 2 }));
      expect(server.charges).toBe(2);
      expect(page.result.current.purchase?.id).toBe("purchase-2");
    } finally {
      restore();
    }
  });

  it.each(["all storage", "writes only", "removals only"])(
    "reuses the key when the browser blocks %s",
    async (blocked) => {
      const storage = Object.getOwnPropertyDescriptor(window, "localStorage");
      const localStorage = window.localStorage;
      Object.defineProperty(window, "localStorage", {
        configurable: true,
        get: () => {
          if (blocked !== "all storage")
            return {
              getItem: localStorage.getItem.bind(localStorage),
              removeItem: (key: string) => {
                if (blocked === "removals only") throw new DOMException("Storage is blocked.", "SecurityError");
                localStorage.removeItem(key);
              },
              setItem: (key: string, value: string) => {
                if (blocked === "writes only") throw new DOMException("Storage is full.", "QuotaExceededError");
                localStorage.setItem(key, value);
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
    },
  );

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
