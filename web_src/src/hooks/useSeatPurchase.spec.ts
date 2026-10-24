import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import type {
  OrganizationsDescribeOrganizationBillingResponse,
  OrganizationsPreviewSeatPurchaseResponse,
  OrganizationsSeatPurchase,
} from "@/api-client";

const {
  organizationsDescribeOrganizationBilling,
  organizationsPreviewSeatPurchase,
  organizationsPurchaseSeats,
  organizationsSyncOrganizationBilling,
} = vi.hoisted(() => ({
  organizationsDescribeOrganizationBilling: vi.fn(),
  organizationsPreviewSeatPurchase: vi.fn(),
  organizationsPurchaseSeats: vi.fn(),
  organizationsSyncOrganizationBilling: vi.fn(),
}));

vi.mock("@/api-client", () => ({
  organizationsDescribeOrganizationBilling,
  organizationsPreviewSeatPurchase,
  organizationsPurchaseSeats,
  organizationsSyncOrganizationBilling,
}));

import { organizationBillingQueryKey, useOrganizationBilling } from "./useOrganizationBilling";
import { useSeatPurchase } from "./useSeatPurchase";

type Billing = OrganizationsDescribeOrganizationBillingResponse;

const initialBilling: Billing = {
  planPerSeat: true,
  planSource: "polar",
  polarSubscriptionStatus: "active",
  purchasedSeats: 4,
  purchasableSeats: 6,
};
const estimate: OrganizationsPreviewSeatPurchaseResponse = {
  purchasedSeats: 4,
  targetSeats: 5,
  proratedChargeCents: "473",
  periodEnd: "2026-11-15T00:00:00Z",
  newMonthlyTotalCents: "5000",
};

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}

function mountPurchase(billing: Billing = initialBilling, enabled = true, pollingEnabled = enabled) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: 3 } },
  });
  queryClient.setQueryData(organizationBillingQueryKey("org-1"), billing);
  function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  }
  const hook = renderHook(
    ({ organizationId }: { organizationId: string }) => {
      const query = useOrganizationBilling(organizationId);
      return useSeatPurchase({ organizationId, billing: query.data, enabled, pollingEnabled });
    },
    { initialProps: { organizationId: "org-1" }, wrapper: Wrapper },
  );
  return { ...hook, queryClient };
}

async function advance(milliseconds: number) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(milliseconds);
  });
}

async function openPreview(hook: ReturnType<typeof mountPurchase>) {
  act(() => hook.result.current.setOpen(true));
  await advance(300);
}

function purchase(status: string): OrganizationsSeatPurchase {
  return { id: "purchase-1", status, fromSeats: 4, toSeats: 5, createdAt: new Date().toISOString() };
}

describe("useSeatPurchase", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
    organizationsPreviewSeatPurchase.mockResolvedValue({ data: estimate });
    organizationsPurchaseSeats.mockResolvedValue({
      data: { purchase: purchase("succeeded"), billing: { ...initialBilling, purchasedSeats: 5 } },
    });
    organizationsSyncOrganizationBilling.mockResolvedValue({ data: initialBilling });
    organizationsDescribeOrganizationBilling.mockResolvedValue({ data: initialBilling });
  });

  afterEach(() => {
    cleanup();
    vi.useRealTimers();
  });

  it("previews one additional seat after 300 milliseconds and keeps the server amounts", async () => {
    const hook = mountPurchase();
    act(() => hook.result.current.setOpen(true));
    expect(hook.result.current.additionalSeats).toBe("1");
    await advance(299);
    expect(organizationsPreviewSeatPurchase).not.toHaveBeenCalled();
    await advance(1);
    expect(organizationsPreviewSeatPurchase).toHaveBeenCalledWith(
      expect.objectContaining({ path: { id: "org-1" }, body: { additionalSeats: 1 } }),
    );
    expect(hook.result.current.preview).toEqual(estimate);
    expect(hook.result.current.canConfirm).toBe(true);
  });

  it("clears estimates immediately and ignores older responses after quantity changes", async () => {
    const older = deferred<{ data: OrganizationsPreviewSeatPurchaseResponse }>();
    const newer = deferred<{ data: OrganizationsPreviewSeatPurchaseResponse }>();
    const hook = mountPurchase();
    await openPreview(hook);
    organizationsPreviewSeatPurchase.mockReturnValueOnce(older.promise).mockReturnValueOnce(newer.promise);
    act(() => hook.result.current.setAdditionalSeats("2"));
    expect(hook.result.current.preview).toBeUndefined();
    expect(hook.result.current.canConfirm).toBe(false);
    await advance(300);
    act(() => hook.result.current.setAdditionalSeats("3"));
    await advance(300);
    await act(async () => newer.resolve({ data: { ...estimate, targetSeats: 7, proratedChargeCents: "1419" } }));
    await act(async () => older.resolve({ data: { ...estimate, targetSeats: 6, proratedChargeCents: "946" } }));
    expect(hook.result.current.preview?.proratedChargeCents).toBe("1419");
    await act(async () => hook.result.current.confirm());
    expect(organizationsPurchaseSeats.mock.calls[0][0].body.additionalSeats).toBe(3);
  });

  it.each(["", "0", "-1", "1.5", "7", "1e2"])("does not preview invalid or over-limit quantity %s", async (value) => {
    const hook = mountPurchase();
    act(() => {
      hook.result.current.setOpen(true);
      hook.result.current.setAdditionalSeats(value);
    });
    await advance(600);
    await act(async () => hook.result.current.confirm());
    expect(organizationsPreviewSeatPurchase).not.toHaveBeenCalled();
    expect(organizationsPurchaseSeats).not.toHaveBeenCalled();
  });

  it("allows any valid positive integer when capacity is unlimited", async () => {
    const hook = mountPurchase({ ...initialBilling, purchasableSeats: undefined });
    act(() => {
      hook.result.current.setOpen(true);
      hook.result.current.setAdditionalSeats("1000");
    });
    await advance(300);
    expect(organizationsPreviewSeatPurchase.mock.calls[0][0].body.additionalSeats).toBe(1000);
  });

  it.each([
    { enabled: false, billing: initialBilling },
    { enabled: true, billing: { ...initialBilling, cancelAtPeriodEnd: true } },
    { enabled: true, billing: { ...initialBilling, purchasableSeats: 0 } },
  ])("does not call purchase APIs when purchasing is blocked (%j)", async ({ enabled, billing }) => {
    const hook = mountPurchase(billing, enabled);
    act(() => hook.result.current.setOpen(true));
    await advance(600);
    await act(async () => hook.result.current.confirm());
    expect(organizationsPreviewSeatPurchase).not.toHaveBeenCalled();
    expect(organizationsPurchaseSeats).not.toHaveBeenCalled();
  });

  it("requires a new preview after the billing baseline changes", async () => {
    const hook = mountPurchase();
    await openPreview(hook);
    await act(async () => {
      hook.queryClient.setQueryData(organizationBillingQueryKey("org-1"), { ...initialBilling, purchasedSeats: 6 });
      await hook.result.current.confirm();
    });
    expect(organizationsPurchaseSeats).not.toHaveBeenCalled();
    hook.rerender({ organizationId: "org-1" });
    expect(hook.result.current.preview).toBeUndefined();
    await act(async () => hook.result.current.confirm());
    expect(organizationsPurchaseSeats).not.toHaveBeenCalled();
    organizationsPreviewSeatPurchase.mockResolvedValue({ data: { ...estimate, purchasedSeats: 6, targetSeats: 7 } });
    await advance(300);
    await act(async () => hook.result.current.confirm());
    expect(organizationsPurchaseSeats.mock.calls[0][0].body.expectedPurchasedSeats).toBe(6);
  });

  it("locks submission synchronously and uses the completed preview baseline", async () => {
    const submitted = deferred<{ data: { billing: Billing; purchase: OrganizationsSeatPurchase } }>();
    organizationsPreviewSeatPurchase.mockResolvedValue({ data: { ...estimate, purchasedSeats: 8 } });
    organizationsPurchaseSeats.mockReturnValue(submitted.promise);
    const hook = mountPurchase();
    await openPreview(hook);
    let first!: Promise<void>;
    act(() => {
      first = hook.result.current.confirm();
      void hook.result.current.confirm();
    });
    expect(hook.result.current.submitting).toBe(true);
    expect(organizationsPurchaseSeats).toHaveBeenCalledTimes(1);
    expect(organizationsPurchaseSeats.mock.calls[0][0].body).toEqual({ additionalSeats: 1, expectedPurchasedSeats: 8 });
    act(() => hook.result.current.setAdditionalSeats("2"));
    expect(hook.result.current.additionalSeats).toBe("1");
    await act(async () => {
      submitted.resolve({ data: { billing: { purchasedSeats: 9 }, purchase: purchase("succeeded") } });
      await first;
    });
    expect(hook.queryClient.getQueryData(organizationBillingQueryKey("org-1"))).toEqual({ purchasedSeats: 9 });
    expect(hook.result.current.purchaseMessage).toBe("Seats added.");
    expect(organizationsSyncOrganizationBilling).not.toHaveBeenCalled();
  });

  it("refreshes authoritative billing when a completed purchase omits billing without purchasing again", async () => {
    organizationsPurchaseSeats.mockResolvedValue({ data: { purchase: purchase("succeeded") } });
    organizationsSyncOrganizationBilling.mockResolvedValue({ data: { purchasedSeats: 5 } });
    const hook = mountPurchase();
    await openPreview(hook);
    await act(async () => hook.result.current.confirm());
    expect(hook.queryClient.getQueryData(organizationBillingQueryKey("org-1"))).toEqual({ purchasedSeats: 5 });
    expect(organizationsPurchaseSeats).toHaveBeenCalledTimes(1);
    expect(organizationsSyncOrganizationBilling).toHaveBeenCalledTimes(1);
    expect(hook.result.current.purchaseMessage).toBe("Seats added.");
  });

  it("retains a confirmed success when the purchase omits billing and the billing refresh fails", async () => {
    const succeeded = purchase("succeeded");
    organizationsPurchaseSeats.mockResolvedValue({ data: { purchase: succeeded } });
    organizationsSyncOrganizationBilling.mockRejectedValue(new Error("Polar is unavailable."));
    organizationsDescribeOrganizationBilling.mockRejectedValue(new Error("Billing is unavailable."));
    const hook = mountPurchase();
    await openPreview(hook);
    await act(async () => hook.result.current.confirm());
    expect(hook.result.current.purchase).toEqual(succeeded);
    expect(hook.result.current.purchaseMessage).toBe("Seats added.");
    expect(organizationsPurchaseSeats).toHaveBeenCalledTimes(1);
    expect(organizationsSyncOrganizationBilling).toHaveBeenCalledTimes(1);
    expect(organizationsDescribeOrganizationBilling).toHaveBeenCalledTimes(1);
  });

  it.each(["failed", "action_required"])(
    "recovers a recorded %s payment after an HTTP error and keeps its server message",
    async (status) => {
      const recorded = { ...purchase(status), message: "Update this card with your bank." };
      organizationsPurchaseSeats.mockRejectedValue({ status: 412, message: "The payment needs attention." });
      organizationsSyncOrganizationBilling.mockResolvedValue({
        data: { ...initialBilling, lastSeatPurchase: recorded },
      });
      const hook = mountPurchase();
      await openPreview(hook);
      await act(async () => hook.result.current.confirm());
      expect(hook.result.current.purchase).toEqual(recorded);
      expect(hook.result.current.purchaseMessage).toBe(recorded.message);
      await advance(10000);
      expect(organizationsPurchaseSeats).toHaveBeenCalledTimes(1);
    },
  );

  it.each(["failed", "action_required"])(
    "retains a %s result returned by the purchase API without scheduling another attempt",
    async (status) => {
      const recorded = { ...purchase(status), message: "Use another payment method." };
      organizationsPurchaseSeats.mockResolvedValue({
        data: { purchase: recorded, billing: { ...initialBilling, lastSeatPurchase: recorded } },
      });
      const hook = mountPurchase();
      await openPreview(hook);
      await act(async () => hook.result.current.confirm());
      expect(hook.result.current.purchase).toEqual(recorded);
      expect(hook.result.current.purchaseMessage).toBe(recorded.message);
      await advance(3000);
      expect(organizationsPreviewSeatPurchase).toHaveBeenCalledTimes(1);
      expect(organizationsPurchaseSeats).toHaveBeenCalledTimes(1);
      expect(organizationsSyncOrganizationBilling).not.toHaveBeenCalled();
    },
  );

  it("reloads billing and previews a conflict without automatically confirming again", async () => {
    organizationsPurchaseSeats.mockRejectedValueOnce({ status: 409, message: "The seat count changed." });
    organizationsSyncOrganizationBilling.mockResolvedValue({ data: { ...initialBilling, purchasedSeats: 7 } });
    const hook = mountPurchase();
    await openPreview(hook);
    organizationsPreviewSeatPurchase.mockResolvedValue({ data: { ...estimate, purchasedSeats: 7, targetSeats: 8 } });
    await act(async () => hook.result.current.confirm());
    expect(hook.result.current.preview).toBeUndefined();
    await advance(300);
    expect(organizationsPreviewSeatPurchase).toHaveBeenCalledTimes(2);
    expect(hook.result.current.preview?.purchasedSeats).toBe(7);
    expect(organizationsPurchaseSeats).toHaveBeenCalledTimes(1);
    organizationsPurchaseSeats.mockResolvedValue({
      data: { purchase: purchase("succeeded"), billing: { purchasedSeats: 8 } },
    });
    await act(async () => hook.result.current.confirm());
    expect(organizationsPurchaseSeats.mock.calls[1][0].body.expectedPurchasedSeats).toBe(7);
  });

  it("shows server renewal freeze messages without retrying the purchase", async () => {
    const message = "Seat purchases are unavailable while your subscription renews. Try again after renewal.";
    organizationsPreviewSeatPurchase.mockRejectedValue({ status: 412, message });
    const hook = mountPurchase();
    await openPreview(hook);
    expect(hook.result.current.previewError).toBe(message);
    expect(hook.result.current.canConfirm).toBe(false);
    organizationsPreviewSeatPurchase.mockResolvedValue({ data: estimate });
    act(() => hook.result.current.setAdditionalSeats("2"));
    await advance(300);
    organizationsPurchaseSeats.mockRejectedValue({ status: 412, message });
    await act(async () => hook.result.current.confirm());
    expect(hook.result.current.purchaseMessage).toBe(message);
    await advance(10000);
    expect(organizationsPurchaseSeats).toHaveBeenCalledTimes(1);
  });

  it("preserves a freeze error when billing still contains an earlier successful purchase", async () => {
    const previous = purchase("succeeded");
    const billing = { ...initialBilling, lastSeatPurchase: previous };
    const message = "Seat purchases are unavailable while your subscription renews.";
    organizationsPurchaseSeats.mockRejectedValue({ status: 412, message });
    organizationsSyncOrganizationBilling.mockResolvedValue({ data: billing });
    const hook = mountPurchase(billing);
    await openPreview(hook);
    await act(async () => hook.result.current.confirm());
    expect(hook.result.current.purchaseMessage).toBe(message);
    expect(organizationsPurchaseSeats).toHaveBeenCalledTimes(1);
  });

  it.each(["pending", "awaiting_confirmation"])(
    "polls %s purchases every three seconds while the dialog is closed and stops on success",
    async (status) => {
      const pending = purchase(status);
      const hook = mountPurchase({ ...initialBilling, lastSeatPurchase: pending });
      organizationsSyncOrganizationBilling.mockResolvedValue({
        data: { ...initialBilling, lastSeatPurchase: pending },
      });
      await advance(2999);
      expect(organizationsSyncOrganizationBilling).not.toHaveBeenCalled();
      await advance(1);
      expect(organizationsSyncOrganizationBilling).toHaveBeenCalledTimes(1);
      act(() => hook.result.current.setOpen(true));
      act(() => hook.result.current.setOpen(false));
      organizationsSyncOrganizationBilling.mockResolvedValue({
        data: { ...initialBilling, purchasedSeats: 5, lastSeatPurchase: purchase("succeeded") },
      });
      await advance(3000);
      expect(hook.result.current.purchaseMessage).toBe("Seats added.");
      await advance(9000);
      expect(organizationsSyncOrganizationBilling).toHaveBeenCalledTimes(2);
    },
  );

  it("keeps payment recovery active when the subscription no longer allows another purchase", async () => {
    const pending = purchase("awaiting_confirmation");
    organizationsSyncOrganizationBilling.mockResolvedValue({ data: { ...initialBilling, lastSeatPurchase: pending } });
    const hook = mountPurchase({ ...initialBilling, lastSeatPurchase: pending }, false, true);
    act(() => hook.result.current.setOpen(true));
    await advance(3000);
    await act(async () => hook.result.current.confirm());
    expect(organizationsSyncOrganizationBilling).toHaveBeenCalledTimes(1);
    expect(organizationsPreviewSeatPurchase).not.toHaveBeenCalled();
    expect(organizationsPurchaseSeats).not.toHaveBeenCalled();
  });

  it("does not restore expired terminal feedback after a pending purchase settles and billing is refreshed", async () => {
    const pending = {
      ...purchase("awaiting_confirmation"),
      createdAt: new Date(Date.now() - 48 * 60 * 60 * 1000).toISOString(),
      message: "Polar has not confirmed this payment yet.",
    };
    const terminal = { ...pending, status: "failed", message: "Polar did not apply the seat purchase." };
    const hook = mountPurchase({ ...initialBilling, lastSeatPurchase: pending });
    expect(hook.result.current.purchaseMessage).toBe(pending.message);
    organizationsSyncOrganizationBilling.mockResolvedValue({ data: { ...initialBilling, lastSeatPurchase: terminal } });
    await advance(3000);
    await act(async () => hook.result.current.refreshBilling());
    expect(hook.result.current.purchase?.status).toBe("failed");
    expect(hook.result.current.purchaseMessage).toBeNull();
  });

  it("does not overlap slow polls and stops starting requests at the two-minute deadline", async () => {
    const pending = purchase("awaiting_confirmation");
    const sync = deferred<{ data: Billing }>();
    organizationsSyncOrganizationBilling.mockReturnValue(sync.promise);
    const hook = mountPurchase({ ...initialBilling, lastSeatPurchase: pending });
    await advance(9000);
    expect(organizationsSyncOrganizationBilling).toHaveBeenCalledTimes(1);
    await advance(111000);
    expect(hook.result.current.timedOut).toBe(true);
    await act(async () => sync.resolve({ data: { ...initialBilling, lastSeatPurchase: pending } }));
    await advance(10000);
    expect(organizationsSyncOrganizationBilling).toHaveBeenCalledTimes(1);
  });

  it("does not restart the same purchase window when the dialog reopens and permits an explicit refresh", async () => {
    const pending = purchase("pending");
    organizationsSyncOrganizationBilling.mockResolvedValue({ data: { ...initialBilling, lastSeatPurchase: pending } });
    const hook = mountPurchase({ ...initialBilling, lastSeatPurchase: pending });
    await advance(60000);
    act(() => hook.result.current.setOpen(true));
    act(() => hook.result.current.setOpen(false));
    await advance(60000);
    expect(hook.result.current.timedOut).toBe(true);
    const polls = organizationsSyncOrganizationBilling.mock.calls.length;
    act(() => hook.result.current.setOpen(true));
    await advance(10000);
    expect(organizationsSyncOrganizationBilling).toHaveBeenCalledTimes(polls);
    organizationsSyncOrganizationBilling.mockResolvedValue({
      data: { ...initialBilling, lastSeatPurchase: purchase("succeeded"), purchasedSeats: 5 },
    });
    await act(async () => hook.result.current.refreshBilling());
    expect(hook.result.current.timedOut).toBe(false);
    expect(hook.result.current.purchaseMessage).toBe("Seats added.");
    expect(organizationsPurchaseSeats).not.toHaveBeenCalled();
  });

  it("stops polling on organization change and page unmount", async () => {
    const pending = purchase("pending");
    const hook = mountPurchase({ ...initialBilling, lastSeatPurchase: pending });
    organizationsSyncOrganizationBilling.mockResolvedValue({ data: { ...initialBilling, lastSeatPurchase: pending } });
    await advance(3000);
    hook.queryClient.setQueryData(organizationBillingQueryKey("org-2"), initialBilling);
    hook.rerender({ organizationId: "org-2" });
    await advance(6000);
    expect(organizationsSyncOrganizationBilling).toHaveBeenCalledTimes(1);
    hook.rerender({ organizationId: "org-1" });
    await advance(3000);
    expect(organizationsSyncOrganizationBilling).toHaveBeenCalledTimes(2);
    hook.unmount();
    await advance(6000);
    expect(organizationsSyncOrganizationBilling).toHaveBeenCalledTimes(2);
  });
});
