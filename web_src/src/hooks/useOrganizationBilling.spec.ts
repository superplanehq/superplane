import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import type { OrganizationsDescribeOrganizationBillingResponse } from "@/api-client";

const synchronize = vi.fn();
vi.mock("@/api-client", () => ({ organizationsSyncOrganizationBilling: synchronize }));

import { organizationBillingQueryKey, syncOrganizationBilling } from "./useOrganizationBilling";

beforeEach(() => synchronize.mockReset());

describe("syncOrganizationBilling", () => {
  it("shares a pending synchronization across billing refresh and purchase polling", async () => {
    let finish!: (response: { data: OrganizationsDescribeOrganizationBillingResponse }) => void;
    synchronize.mockImplementation(
      () => new Promise<{ data: OrganizationsDescribeOrganizationBillingResponse }>((resolve) => (finish = resolve)),
    );
    const client = new QueryClient();
    const refresh = syncOrganizationBilling("org-1", client);
    const purchasePoll = syncOrganizationBilling("org-1", client);
    expect(synchronize).toHaveBeenCalledTimes(1);

    const billing = { purchasedSeats: 4, hasPaymentMethod: true };
    finish({ data: billing });
    expect(await refresh).toEqual(billing);
    expect(await purchasePoll).toEqual(billing);
    expect(client.getQueryData(organizationBillingQueryKey("org-1"))).toEqual(billing);

    synchronize.mockResolvedValue({ data: { purchasedSeats: 5 } });
    await syncOrganizationBilling("org-1", client);
    expect(synchronize).toHaveBeenCalledTimes(2);
    expect(client.getQueryData(organizationBillingQueryKey("org-1"))).toEqual({ purchasedSeats: 5 });
  });

  it("keeps independent organizations and query clients isolated", async () => {
    const client = new QueryClient();
    const otherClient = new QueryClient();
    synchronize.mockImplementation(async ({ path }: { path: { id: string } }) => ({
      data: { planName: path.id },
    }));
    await Promise.all([
      syncOrganizationBilling("org-1", client),
      syncOrganizationBilling("org-2", client),
      syncOrganizationBilling("org-1", otherClient),
    ]);
    expect(synchronize).toHaveBeenCalledTimes(3);
    expect(client.getQueryData(organizationBillingQueryKey("org-1"))).toEqual({ planName: "org-1" });
    expect(client.getQueryData(organizationBillingQueryKey("org-2"))).toEqual({ planName: "org-2" });
    expect(otherClient.getQueryData(organizationBillingQueryKey("org-1"))).toEqual({ planName: "org-1" });
  });

  it.each(["request failure", "missing billing"])(
    "preserves cached billing after %s and permits a retry",
    async (failure) => {
      const client = new QueryClient();
      const billing = { purchasedSeats: 3 };
      client.setQueryData(organizationBillingQueryKey("org-1"), billing);
      if (failure === "request failure") synchronize.mockRejectedValueOnce(new Error("Polar unavailable"));
      else synchronize.mockResolvedValueOnce({});
      await expect(syncOrganizationBilling("org-1", client)).rejects.toThrow();
      expect(client.getQueryData(organizationBillingQueryKey("org-1"))).toEqual(billing);

      synchronize.mockResolvedValue({ data: { purchasedSeats: 4 } });
      await syncOrganizationBilling("org-1", client);
      expect(client.getQueryData(organizationBillingQueryKey("org-1"))).toEqual({ purchasedSeats: 4 });
    },
  );
});
