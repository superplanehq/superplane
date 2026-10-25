import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "bun:test";

import type { OrganizationsDescribeOrganizationBillingResponse } from "@/api-client";
import { client } from "@/api-client/client.gen";
import { organizationBillingQueryKey, useOrganizationBilling } from "./useOrganizationBilling";
import { useSeatPurchase } from "./useSeatPurchase";

describe("seat purchase HTTP recovery", () => {
  const previousConfig = client.getConfig();

  afterEach(() => {
    cleanup();
    client.setConfig({ ...previousConfig, fetch: previousConfig.fetch });
    vi.useRealTimers();
  });

  it("uses an HTTP 409 response to refresh the baseline and require a new confirmation", async () => {
    vi.useFakeTimers();
    let purchasedSeats = 4;
    let previewRequests = 0;
    const purchaseRequests: unknown[] = [];
    const transport: typeof fetch = async (input) => {
      const request = input instanceof Request ? input : new Request(input);
      const path = new URL(request.url).pathname;
      let body: unknown;
      let status = 200;
      if (path.endsWith("/seats/preview")) {
        previewRequests += 1;
        body = { purchasedSeats, targetSeats: purchasedSeats + 1, proratedChargeCents: "500" };
      } else if (path.endsWith("/seats/purchase")) {
        purchaseRequests.push(await request.json());
        purchasedSeats = 7;
        body = { message: "The seat count changed. Preview this purchase again." };
        status = 409;
      } else if (path.endsWith("/billing/sync")) {
        body = { purchasedSeats, purchasableSeats: 10 };
      } else {
        throw new Error(`Unexpected billing request: ${path}`);
      }
      return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
    };
    client.setConfig({ baseUrl: "http://localhost", fetch: transport });
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
    queryClient.setQueryData<OrganizationsDescribeOrganizationBillingResponse>(organizationBillingQueryKey("org-1"), {
      purchasedSeats: 4,
      purchasableSeats: 10,
    });
    function Wrapper({ children }: { children: ReactNode }) {
      return createElement(QueryClientProvider, { client: queryClient }, children);
    }
    const { result } = renderHook(
      () => {
        const billing = useOrganizationBilling("org-1");
        return useSeatPurchase({ organizationId: "org-1", billing: billing.data, enabled: true });
      },
      { wrapper: Wrapper },
    );
    act(() => result.current.setOpen(true));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300);
    });
    expect(result.current.preview?.purchasedSeats).toBe(4);
    await act(async () => result.current.confirm());
    expect(result.current.preview).toBeUndefined();
    expect(purchaseRequests).toEqual([
      { additionalSeats: 1, expectedPurchasedSeats: 4, operationKey: expect.stringMatching(/^[0-9a-f-]{36}$/) },
    ]);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300);
    });
    expect(previewRequests).toBe(2);
    expect(result.current.preview?.purchasedSeats).toBe(7);
    expect(result.current.canConfirm).toBe(true);
    expect(purchaseRequests).toHaveLength(1);
  });
});
