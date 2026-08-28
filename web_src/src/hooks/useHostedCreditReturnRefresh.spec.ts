import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { rememberHostedCreditGrantSnapshot } from "@/lib/hostedCredit";

import { useHostedCreditReturnRefresh } from "./useHostedCreditReturnRefresh";

describe("useHostedCreditReturnRefresh", () => {
  afterEach(() => {
    sessionStorage.clear();
  });

  it("keeps refreshing until the grant total increases", async () => {
    rememberHostedCreditGrantSnapshot("org-1", 2500);
    const refetch = vi.fn().mockResolvedValue(undefined);
    const { result, rerender } = renderHook(
      ({ grantTotalCents }) =>
        useHostedCreditReturnRefresh({
          organizationId: "org-1",
          creditAdded: true,
          grantTotalCents,
          refetch,
        }),
      { initialProps: { grantTotalCents: 2500 } },
    );

    await waitFor(() => expect(refetch).toHaveBeenCalled());
    expect(result.current).toBe("refreshing");

    rerender({ grantTotalCents: 5000 });
    await waitFor(() => expect(result.current).toBe("added"));
  });
});
