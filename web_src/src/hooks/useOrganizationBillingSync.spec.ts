import { act, renderHook, waitFor } from "@testing-library/react";
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

    await act(async () => {});
    expect(sync.mock.calls.length).toBeGreaterThanOrEqual(1);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(BILLING_SUBSCRIPTION_SYNC_INTERVAL_MS);
    });
    const callsAfterPoll = sync.mock.calls.length;
    expect(callsAfterPoll).toBeGreaterThan(1);

    rerender({ creditPurchaseAllowed: true });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(BILLING_SUBSCRIPTION_SYNC_INTERVAL_MS);
    });
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
  it("does not overlap slow sync and refetch cycles", async () => {
    vi.useFakeTimers();
    let release!: () => void;
    const sync = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          release = resolve;
        }),
    );
    const refetch = vi.fn().mockResolvedValue(undefined);
    const { unmount } = renderHook(() =>
      useOrganizationBillingSync({
        organizationId: "org-slow",
        subscribed: true,
        creditPurchaseAllowed: false,
        returnRefreshPending: true,
        sync,
        refetch,
      }),
    );
    await vi.advanceTimersByTimeAsync(6000);
    expect(sync).toHaveBeenCalledTimes(1);
    expect(refetch).not.toHaveBeenCalled();
    release();
    await Promise.resolve();
    await Promise.resolve();
    expect(refetch).toHaveBeenCalledTimes(1);
    unmount();
  });

  it("stops polling an unconfirmed purchase after the timeout", async () => {
    vi.useFakeTimers();
    const sync = vi.fn().mockResolvedValue(undefined);
    const refetch = vi.fn().mockResolvedValue(undefined);
    const { unmount } = renderHook(() =>
      useOrganizationBillingSync({
        organizationId: "org-timeout",
        subscribed: false,
        creditPurchaseAllowed: true,
        returnRefreshPending: true,
        sync,
        refetch,
      }),
    );
    await act(async () => {});
    for (let i = 0; i < 31; i++) {
      await act(async () => {
        await vi.advanceTimersByTimeAsync(2000);
      });
    }
    const calls = refetch.mock.calls.length;
    expect(calls).toBeGreaterThan(1);
    await vi.advanceTimersByTimeAsync(10000);
    expect(refetch).toHaveBeenCalledTimes(calls);
    unmount();
  });
});
