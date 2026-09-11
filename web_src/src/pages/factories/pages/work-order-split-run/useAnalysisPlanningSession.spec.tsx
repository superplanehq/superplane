import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";

import { useAnalysisPlanningSession } from "./useAnalysisPlanningSession";

function wrapper({ children }: { children: ReactNode }) {
  return (
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      {children}
    </QueryClientProvider>
  );
}

describe("useAnalysisPlanningSession", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("keeps chat closed when the draft has no analysis session", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        status: 404,
        ok: false,
        json: async () => ({}),
      }),
    );

    const { result } = renderHook(
      () =>
        useAnalysisPlanningSession({
          organizationId: "org-1",
          factoryId: "factory-1",
          workOrderId: "order-1",
          enabled: true,
          canUpdate: true,
        }),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.showChat).toBe(false);
    });
    expect(result.current.organizationId).toBe("org-1");
    expect(result.current.canSend).toBe(false);
  });
});
