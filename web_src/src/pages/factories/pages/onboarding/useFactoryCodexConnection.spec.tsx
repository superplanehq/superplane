import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { factoriesDescribeFactoryCodexConnection, factoriesStartFactoryCodexConnection } from "@/api-client";

import { useFactoryCodexConnection } from "./useFactoryCodexConnection";

vi.mock("@/api-client", () => ({
  factoriesDescribeFactoryCodexConnection: vi.fn(),
  factoriesStartFactoryCodexConnection: vi.fn(),
  factoriesCancelFactoryCodexConnection: vi.fn(),
}));

function wrapper({ children }: { children: ReactNode }) {
  return (
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      {children}
    </QueryClientProvider>
  );
}

describe("useFactoryCodexConnection", () => {
  beforeEach(() => {
    vi.mocked(factoriesDescribeFactoryCodexConnection).mockReset();
    vi.mocked(factoriesStartFactoryCodexConnection).mockReset();
    vi.mocked(factoriesStartFactoryCodexConnection).mockResolvedValue({ data: {} } as never);
  });

  it("refreshes a ready legacy connection with no model catalog", async () => {
    vi.mocked(factoriesDescribeFactoryCodexConnection).mockResolvedValue({
      data: { connection: { status: "ready", availableModels: [] } },
    } as never);

    renderHook(() => useFactoryCodexConnection("org-1", "factory-1", true), { wrapper });

    await waitFor(() => expect(factoriesStartFactoryCodexConnection).toHaveBeenCalledTimes(1));
    expect(factoriesStartFactoryCodexConnection).toHaveBeenCalledWith(
      expect.objectContaining({
        path: { factoryId: "factory-1" },
        body: { refreshCatalog: true },
      }),
    );
  });

  it("does not refresh a connection that already has models", async () => {
    vi.mocked(factoriesDescribeFactoryCodexConnection).mockResolvedValue({
      data: { connection: { status: "ready", availableModels: ["gpt-5.6-sol"] } },
    } as never);

    const result = renderHook(() => useFactoryCodexConnection("org-1", "factory-1", true), { wrapper });
    await waitFor(() => expect(result.result.current.query.isSuccess).toBe(true));

    expect(factoriesStartFactoryCodexConnection).not.toHaveBeenCalled();
  });
});
