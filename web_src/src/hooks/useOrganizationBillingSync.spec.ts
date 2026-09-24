import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { BILLING_SUBSCRIPTION_SYNC_INTERVAL_MS, useOrganizationBillingSync } from "./useOrganizationBillingSync";

describe("useOrganizationBillingSync", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("syncs Polar on mount", async () => {
    const sync = vi.fn().mockResolvedValue(undefined);
    const refetch = vi.fn().mockResolvedValue(undefined);

    renderHook(() =>
      useOrganizationBillingSync({
        organizationId: "org-1",
        subscribed: false,
        creditPurchaseAllowed: false,
        sync,
        refetch,
      }),
    );

    await waitFor(() => expect(sync).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(refetch).toHaveBeenCalledTimes(1));
  });

  it("keeps polling after subscribe until the plan is Business", async () => {
    vi.useFakeTimers();
    const sync = vi.fn().mockResolvedValue(undefined);
    const refetch = vi.fn().mockResolvedValue(undefined);
    const { rerender } = renderHook(
      ({ creditPurchaseAllowed }) =>
        useOrganizationBillingSync({
          organizationId: "org-1",
          subscribed: true,
          creditPurchaseAllowed,
          sync,
          refetch,
        }),
      { initialProps: { creditPurchaseAllowed: false } },
    );

    await vi.runOnlyPendingTimersAsync();
    expect(sync.mock.calls.length).toBeGreaterThanOrEqual(1);

    await vi.advanceTimersByTimeAsync(BILLING_SUBSCRIPTION_SYNC_INTERVAL_MS);
    const callsAfterPoll = sync.mock.calls.length;
    expect(callsAfterPoll).toBeGreaterThan(1);

    rerender({ creditPurchaseAllowed: true });
    await vi.advanceTimersByTimeAsync(BILLING_SUBSCRIPTION_SYNC_INTERVAL_MS);
    expect(sync).toHaveBeenCalledTimes(callsAfterPoll);
  });

  it("refetches local billing when Polar sync fails", async () => {
    const sync = vi.fn().mockRejectedValue(new Error("polar down"));
    const refetch = vi.fn().mockResolvedValue(undefined);

    renderHook(() =>
      useOrganizationBillingSync({
        organizationId: "org-1",
        subscribed: false,
        creditPurchaseAllowed: false,
        sync,
        refetch,
      }),
    );

    await waitFor(() => expect(sync).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(refetch).toHaveBeenCalledTimes(1));
  });
});
